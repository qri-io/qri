package lib

import (
	"context"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/preview"
	"github.com/qri-io/qri/automation/hook"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/dsref"
)

// AutomationMethods groups together methods for automations
type AutomationMethods struct {
	d dispatcher
}

// Name returns the name of this method group
func (m AutomationMethods) Name() string {
	return "automation"
}

// Attributes defines attributes for each method
func (m AutomationMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"apply":  {Endpoint: AEApply, HTTPVerb: "POST"},
		"deploy": {Endpoint: AEDeploy, HTTPVerb: "POST"},
	}
}

// ApplyParams are parameters for the apply command
type ApplyParams struct {
	Ref       string             `json:"ref"`
	Transform *dataset.Transform `json:"transform"`
	Secrets   map[string]string  `json:"secrets"`
	Wait      bool               `json:"wait"`
	// TODO(arqu): substitute with websockets when working over the wire
	ScriptOutput io.Writer `json:"-"`
	Hooks        []hook.Hook
}

// Validate returns an error if ApplyParams fields are in an invalid state
func (p *ApplyParams) Validate() error {
	if p.Ref == "" && p.Transform == nil {
		return fmt.Errorf("one or both of Reference, Transform are required")
	}
	return nil
}

// ApplyResult is the result of an apply command
type ApplyResult struct {
	Data  *dataset.Dataset
	RunID string `json:"runID"`
}

// Apply runs a transform script
func (m AutomationMethods) Apply(ctx context.Context, p *ApplyParams) (*ApplyResult, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "apply"), p)
	if res, ok := got.(*ApplyResult); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// DeployParams are parameters for the deploy command
type DeployParams struct {
	Apply    bool // when Apply is true, run the workflow after updating the dataset and workflow
	Workflow *workflow.Workflow
	Dataset  *dataset.Dataset
}

// Validate returns an error if DeployParams fields are in an invalid state
func (p *DeployParams) Validate() error {
	if p.Workflow == nil {
		return fmt.Errorf("deploy: workflow not set")
	}
	if p.Workflow.DatasetID == "" && p.Dataset == nil {
		return fmt.Errorf("deploy: either dataset or workflow.DatasetID are required")
	}
	if p.Dataset != nil && p.Workflow.DatasetID != "" && p.Workflow.DatasetID != p.Dataset.ID {
		return fmt.Errorf("deploy: dataset id and the dataset id in the workflow must match")
	}
	return nil
}

// DeployResponse is the result of a deploy command
type DeployResponse struct {
	RunID    string             `json:"runID,omitempty"`
	Workflow *workflow.Workflow `json:"workflow"`
}

// Deploy adds or updates a workflow
func (m AutomationMethods) Deploy(ctx context.Context, p *DeployParams) (*DeployResponse, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "deploy"), p)
	if res, ok := got.(*DeployResponse); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Implementations for automation methods follow

// automationImpl holds the method implementations for automations
type automationImpl struct{}

// Apply runs a transform script
func (automationImpl) Apply(scope scope, p *ApplyParams) (*ApplyResult, error) {
	ctx := scope.Context()

	var err error
	ref := dsref.Ref{}
	if p.Ref != "" {
		scope.EnableWorkingDir(true)
		ref, _, err = scope.ParseAndResolveRef(ctx, p.Ref)
		if err != nil {
			return nil, err
		}
	}

	ds := &dataset.Dataset{}
	if !ref.IsEmpty() {
		ds.Name = ref.Name
		ds.Peername = ref.Username
	}
	if p.Transform != nil {
		ds.Transform = p.Transform
		ds.Transform.OpenScriptFile(ctx, scope.Filesystem())
	}

	wf := &workflow.Workflow{}
	if p.Hooks != nil {
		wf.Hooks = p.Hooks
	}

	runID, err := scope.AutomationOrchestrator().ApplyWorkflow(ctx, p.Wait, p.ScriptOutput, wf, ds, p.Secrets)
	if err != nil {
		return nil, err
	}

	res := &ApplyResult{}
	if p.Wait {
		ds, err := preview.Create(ctx, ds)
		if err != nil {
			return nil, err
		}
		res.Data = ds
	}
	res.RunID = runID
	return res, nil
}

// Deploy adds or updates a Dataset, creates or updates an associated Workflow, and, if deployParams.Apply is true, immediately runs the Workflow
func (automationImpl) Deploy(scope scope, p *DeployParams) (*DeployResponse, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (inst *Instance) run(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow, runID string) error {
	ctxWithProfile := profile.AddIDToContext(ctx, w.OwnerID.String())
	scope, err := newScope(ctxWithProfile, inst, "local")
	if err != nil {
		return err
	}
	p := &SaveParams{
		Dataset: &dataset.Dataset{
			ID: w.DatasetID,
			Commit: &dataset.Commit{
				RunID: runID,
			},
		},
		Apply: true,
	}
	dImpl := &datasetImpl{}
	_, err = dImpl.Save(scope, p)
	return err
}

func (inst *Instance) apply(ctx context.Context, wait bool, runID string, wf *workflow.Workflow, ds *dataset.Dataset, secrets map[string]string) error {
	ctxWithProfile := profile.AddIDToContext(ctx, wf.OwnerID.String())
	scope, err := newScope(ctxWithProfile, inst, "local")
	if err != nil {
		return err
	}

	transformer := transform.NewTransformer(scope.AppContext(), scope.Loader(), scope.Bus())
	return transformer.Apply(ctx, ds, runID, wait, nil, secrets)
}
