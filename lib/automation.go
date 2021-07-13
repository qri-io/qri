package lib

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/preview"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/transform"
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
	Hooks        []map[string]interface{}
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
	Run      bool // when Run is true, run the workflow after updating the dataset and workflow
	Workflow *workflow.Workflow
	Dataset  *dataset.Dataset
}

// Validate returns an error if DeployParams fields are in an invalid state
func (p *DeployParams) Validate() error {
	if p.Workflow == nil {
		return fmt.Errorf("deploy: workflow required")
	}
	if p.Dataset == nil {
		return fmt.Errorf("deploy: dataset required")
	}
	if p.Dataset.Name == "" || p.Dataset.Peername == "" {
		return fmt.Errorf("deploy: dataset name and peername required")
	}
	return nil
}

// Deploy adds or updates a workflow
func (m AutomationMethods) Deploy(ctx context.Context, p *DeployParams) error {
	_, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "deploy"), p)
	return dispatchReturnError(nil, err)
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
func (automationImpl) Deploy(scope scope, p *DeployParams) error {
	log.Debugw("deploy", "dataset name", p.Dataset.Name, "peername", p.Dataset.Peername, "workflow id", p.Workflow.ID)
	if p.Workflow.ID != "" {
		wf, err := scope.AutomationOrchestrator().GetWorkflow(p.Workflow.ID)
		if err != nil {
			return nil
		}
		if p.Workflow.DatasetID != wf.DatasetID {
			return fmt.Errorf("deploy: given workflow and workflow on record have different DatasetIDs")
		}
		if p.Workflow.OwnerID != wf.OwnerID {
			return fmt.Errorf("deploy: given workflow and workflow on record have different OwnerIDs")
		}
	}
	go deploy(scope, p)
	return nil
}

func deploy(scope scope, p *DeployParams) {
	ctx := profile.AddIDToContext(scope.AppContext(), scope.ActiveProfile().ID.String())
	vi := dsref.ConvertDatasetToVersionInfo(p.Dataset)
	ref := vi.SimpleRef().String()
	deployPayload := event.DeployEvent{
		Ref:        ref,
		DatasetID:  p.Dataset.ID,
		WorkflowID: p.Workflow.ID.String(),
	}
	log.Debugw("deploy started", "payload", deployPayload)
	go func() {
		if err := scope.Bus().PublishID(ctx, event.ETAutomationDeployStart, ref, deployPayload); err != nil {
			log.Debug(err)
		}
	}()
	rollback := true
	defer func() {
		if rollback {
			rp := &RemoveParams{
				Ref: ref,
				Revision: &dsref.Rev{
					Field: "ds",
					Gen:   1,
				},
			}
			if _, err := (datasetImpl{}).Remove(scope, rp); err != nil {
				log.Debugw("deploy rollback", "error", err)
			}
		}
	}()

	go func() {
		if err := scope.Bus().PublishID(scope.Context(), event.ETAutomationDeploySaveDatasetStart, ref, deployPayload); err != nil {
			log.Debug(err)
		}
	}()

	saveParams := &SaveParams{
		Ref:     vi.SimpleRef().String(),
		Dataset: p.Dataset,
		Apply:   false,
	}
	// TODO(ramfox): bandaid! remove when save can handle having saving a dataset with no body/structure
	if p.Dataset.ID == "" && p.Dataset.BodyFile() == nil {
		saveParams.Apply = true
	}
	ds, err := datasetImpl{}.Save(scope, saveParams)
	if err != nil && !errors.Is(err, dsfs.ErrNoChanges) {
		log.Debugw("deploy save dataset", "error", err)
		deployPayload.Error = err.Error()
		go func() {
			if err := scope.Bus().PublishID(scope.Context(), event.ETAutomationDeployEnd, ref, deployPayload); err != nil {
				log.Debug(err)
			}
		}()
		return
	}

	deployPayload.DatasetID = ds.ID
	go func() {
		if err := scope.Bus().PublishID(scope.Context(), event.ETAutomationDeploySaveDatasetEnd, ref, deployPayload); err != nil {
			log.Debug(err)
		}
	}()

	wf := p.Workflow.Copy()
	wf.DatasetID = ds.ID
	wf.OwnerID = scope.ActiveProfile().ID

	go func() {
		if err := scope.Bus().PublishID(scope.Context(), event.ETAutomationDeploySaveWorkflowStart, ref, deployPayload); err != nil {
			log.Debug(err)
		}
	}()

	wf, err = scope.AutomationOrchestrator().SaveWorkflow(wf)
	if err != nil {
		log.Debugw("deploy save workflow", "error", err)
		deployPayload.Error = err.Error()
		go func() {
			if err := scope.Bus().PublishID(scope.Context(), event.ETAutomationDeployEnd, ref, deployPayload); err != nil {
				log.Debug(err)
			}
		}()
		return
	}

	deployPayload.WorkflowID = wf.ID.String()
	go func() {
		if err := scope.Bus().PublishID(scope.Context(), event.ETAutomationDeploySaveWorkflowEnd, ref, deployPayload); err != nil {
			log.Debug(err)
		}
	}()

	if p.Run && !saveParams.Apply {
		runID := run.NewID()

		deployPayload.RunID = runID
		go func() {
			if err := scope.Bus().PublishID(scope.Context(), event.ETAutomationDeployRun, ref, deployPayload); err != nil {
				log.Debug(err)
			}
		}()

		err := scope.AutomationOrchestrator().RunWorkflow(scope.Context(), wf.ID, runID)
		if err != nil {
			log.Debugw("deploy run workflow", "error", err)
			deployPayload.Error = err.Error()
			go func() {
				if err := scope.Bus().PublishID(scope.Context(), event.ETAutomationDeployEnd, ref, deployPayload); err != nil {
					log.Debug(err)
				}
			}()
			return
		}

	}
	log.Debug("deploy ended")
	go func() {
		if err := scope.Bus().PublishID(scope.Context(), event.ETAutomationDeployEnd, ref, deployPayload); err != nil {
			log.Debug(err)
		}
	}()
	rollback = false
}

func (inst *Instance) run(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow, runID string) error {
	ctxWithProfile := profile.AddIDToContext(ctx, w.OwnerID.String())
	scope, err := newScope(ctxWithProfile, inst, "local")
	if err != nil {
		return err
	}
	ref := &dsref.Ref{InitID: w.DatasetID}
	_, err = scope.ResolveReference(ctx, ref)
	if err != nil {
		return fmt.Errorf("run error: %w", err)
	}
	p := &SaveParams{
		Ref: ref.Human(),
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
