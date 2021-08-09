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
	qhttp "github.com/qri-io/qri/lib/http"
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
		"apply":    {Endpoint: qhttp.AEApply, HTTPVerb: "POST"},
		"deploy":   {Endpoint: qhttp.AEDeploy, HTTPVerb: "POST", DefaultSource: "local"},
		"run":      {Endpoint: qhttp.AERun, HTTPVerb: "POST"},
		"workflow": {Endpoint: qhttp.AEWorkflow, HTTPVerb: "POST"},
		"remove":   {Endpoint: qhttp.AERemoveWorkflow, HTTPVerb: "POST"},
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

// RunParams are parameters for the run command
type RunParams struct {
	Ref        string `json:"ref"`
	InitID     string `json:"initID"`
	WorkflowID string `json:"workflowID"`
	RunID      string `json:"runID"`
}

// Validate returns an error if RunParams fields are in an invalid state
func (p *RunParams) Validate() error {
	if p.WorkflowID == "" && p.InitID == "" && p.Ref == "" {
		return fmt.Errorf("run params: workflow id, init id, or ref required")
	}
	if (p.WorkflowID != "" && p.InitID != "") || (p.WorkflowID != "" && p.Ref != "") || (p.InitID != "" && p.Ref != "") {
		return fmt.Errorf("run params: only one of workflow id, init id, or ref needed")
	}
	return nil
}

// Run manually runs a workflow
func (m AutomationMethods) Run(ctx context.Context, p *RunParams) (string, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "run"), p)
	if res, ok := got.(string); ok {
		return res, err
	}
	return "", dispatchReturnError(got, err)
}

// WorkflowParams are parameters for the Workflow command
type WorkflowParams struct {
	WorkflowID string `json:"workflowID"`
	InitID     string `json:"initID"`
	Ref        string `json:"ref"`
}

// Validate returns an error if WorkflowParams fields are in an invalid state
func (p *WorkflowParams) Validate() error {
	if p.WorkflowID == "" && p.InitID == "" && p.Ref == "" {
		return fmt.Errorf("workflow params: workflow id, init id, or ref must be provided")
	}
	if (p.WorkflowID != "" && p.InitID != "") || (p.WorkflowID != "" && p.Ref != "") || (p.InitID != "" && p.Ref != "") {
		return fmt.Errorf("workflow params: only one of workflow id, init id, or ref needed")
	}

	return nil
}

// Workflow fetches a workflow
func (m AutomationMethods) Workflow(ctx context.Context, p *WorkflowParams) (*workflow.Workflow, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "workflow"), p)
	if res, ok := got.(*workflow.Workflow); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Remove removes a workflow
func (m AutomationMethods) Remove(ctx context.Context, p *WorkflowParams) error {
	_, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "remove"), p)
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
		wf, err := scope.AutomationOrchestrator().GetWorkflow(scope.Context(), p.Workflow.ID)
		if err != nil {
			return fmt.Errorf("deploy: %w", err)
		}
		if p.Workflow.InitID != wf.InitID {
			return fmt.Errorf("deploy: given workflow and workflow on record have different InitIDs")
		}
		if p.Workflow.OwnerID != wf.OwnerID {
			return fmt.Errorf("deploy: given workflow and workflow on record have different OwnerIDs")
		}
	}

	// Because deploy runs as a background task, re-root execution context atop
	// the application context
	log.Debugw("app context", "ctx", scope.AppContext())
	scope = scope.ReplaceParentContext(scope.AppContext())
	// TODO(ramfox): if we decide that you can interact with the automation subsystem when
	// qri connect is NOT running, we need a `wait` flag in DeployParams, that, when `true`,
	// does NOT deploy in a go routine
	go deploy(scope, p)
	return nil
}

func deploy(scope scope, p *DeployParams) {
	vi := dsref.ConvertDatasetToVersionInfo(p.Dataset)
	ref := vi.SimpleRef().String()
	deployPayload := event.DeployEvent{
		Ref:        ref,
		InitID:     p.Dataset.ID,
		WorkflowID: p.Workflow.ID.String(),
	}

	log.Debugw("deploy started", "ref", vi.SimpleRef().String(), "payload", deployPayload)
	scope.sendEvent(event.ETAutomationDeployStart, ref, deployPayload)

	go scope.sendEvent(event.ETAutomationDeploySaveDatasetStart, ref, deployPayload)

	changesSaved := false
	rollback := true
	defer func() {
		if rollback && changesSaved {
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
		scope.sendEvent(event.ETAutomationDeployEnd, ref, deployPayload)
		return
	}
	changesSaved = !errors.Is(err, dsfs.ErrNoChanges)
	deployPayload.InitID = ds.ID
	go scope.sendEvent(event.ETAutomationDeploySaveDatasetEnd, ref, deployPayload)

	wf := p.Workflow.Copy()
	wf.InitID = ds.ID
	wf.OwnerID = scope.ActiveProfile().ID

	go scope.sendEvent(event.ETAutomationDeploySaveWorkflowStart, ref, deployPayload)

	wf, err = scope.AutomationOrchestrator().SaveWorkflow(scope.Context(), wf)
	if err != nil {
		log.Debugw("deploy save workflow", "error", err)
		deployPayload.Error = err.Error()
		scope.sendEvent(event.ETAutomationDeployEnd, ref, deployPayload)
		return
	}

	deployPayload.WorkflowID = wf.ID.String()
	go scope.sendEvent(event.ETAutomationDeploySaveWorkflowEnd, ref, deployPayload)

	if p.Run && !saveParams.Apply {
		runID := run.NewID()

		deployPayload.RunID = runID
		go scope.sendEvent(event.ETAutomationDeployRun, ref, deployPayload)

		_, err := scope.AutomationOrchestrator().RunWorkflow(scope.Context(), wf.ID, runID)
		if err != nil {
			log.Debugw("deploy run workflow", "error", err)
			deployPayload.Error = err.Error()
			scope.sendEvent(event.ETAutomationDeployEnd, ref, deployPayload)
			return
		}
	}

	log.Debug("deploy ended")
	scope.sendEvent(event.ETAutomationDeployEnd, ref, deployPayload)
	rollback = false
}

// Run manually runs a workflow
func (automationImpl) Run(scope scope, p *RunParams) (string, error) {
	if p.WorkflowID == "" {
		if p.Ref != "" && p.InitID == "" {
			ref, err := dsref.Parse(p.Ref)
			if err != nil {
				return "", err
			}
			if _, err := scope.ResolveReference(scope.Context(), &ref); err != nil {
				return "", err
			}
			p.InitID = ref.InitID
		}
		wf, err := scope.AutomationOrchestrator().GetWorkflowByInitID(scope.Context(), p.InitID)
		if err != nil {
			return "", err
		}
		p.WorkflowID = wf.WorkflowID()
	}
	return scope.AutomationOrchestrator().RunWorkflow(scope.AppContext(), workflow.ID(p.WorkflowID), p.RunID)
}

// Workflow fetches a workflow by the workflow or dataset id
func (automationImpl) Workflow(scope scope, p *WorkflowParams) (*workflow.Workflow, error) {
	if p.WorkflowID != "" {
		return scope.AutomationOrchestrator().GetWorkflow(scope.Context(), workflow.ID(p.WorkflowID))
	}
	if p.Ref != "" && p.InitID == "" {
		ref, err := dsref.Parse(p.Ref)
		if err != nil {
			return nil, err
		}
		if _, err := scope.ResolveReference(scope.Context(), &ref); err != nil {
			return nil, err
		}
		p.InitID = ref.InitID
	}
	return scope.AutomationOrchestrator().GetWorkflowByInitID(scope.Context(), p.InitID)
}

// Remove removes a workflow by the workflow or dataset id
func (automationImpl) Remove(scope scope, p *WorkflowParams) error {
	if p.WorkflowID == "" {
		if p.Ref != "" && p.InitID == "" {
			ref, err := dsref.Parse(p.Ref)
			if err != nil {
				return err
			}
			if _, err := scope.ResolveReference(scope.Context(), &ref); err != nil {
				return err
			}
			p.InitID = ref.InitID
		}
		wf, err := scope.AutomationOrchestrator().GetWorkflowByInitID(scope.Context(), p.InitID)
		if err != nil {
			return err
		}
		p.WorkflowID = wf.WorkflowID()
	}

	return scope.AutomationOrchestrator().RemoveWorkflow(scope.Context(), workflow.ID(p.WorkflowID))
}

func (inst *Instance) run(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow, runID string) error {
	scope, err := newScopeFromWorkflow(ctx, inst, w)
	if err != nil {
		return err
	}
	ref := &dsref.Ref{InitID: w.InitID}
	_, err = scope.ResolveReference(ctx, ref)
	if err != nil {
		return fmt.Errorf("run error: %w", err)
	}
	p := &SaveParams{
		Ref: ref.Human(),
		Dataset: &dataset.Dataset{
			ID: w.InitID,
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
	scope, err := newScopeFromWorkflow(ctx, inst, wf)
	if err != nil {
		return err
	}

	transformer := transform.NewTransformer(scope.AppContext(), scope.Loader(), scope.Bus())
	return transformer.Apply(ctx, ds, runID, wait, nil, secrets)
}
