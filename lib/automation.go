package lib

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/preview"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/scheduler"
	"github.com/qri-io/qri/automation/transform"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
)

// AutomationMethods groups together methods for automations
// TODO(b5): expand apply methods:
//   automation.runs              // list automation runs
//   automation.run               // get automation run log
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
		"apply":          {AEApply, "POST"},
		"deployworkflow": {AEDeployWorkflow, "POST"},
		"getworkflow":    {AEGetWorkflow, "POST"},
		"listworkflows":  {AEListWorkflows, "POST"},
	}
}

// ApplyParams are parameters for the apply command
type ApplyParams struct {
	Ref       string
	Transform *dataset.Transform
	Secrets   map[string]string
	Wait      bool
	// TODO(arqu): substitute with websockets when working over the wire
	ScriptOutput io.Writer `json:"-"`
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

type ListWorkflowParams struct {
	Offset int
	Limit  int
}

func (m AutomationMethods) ListWorkflows(ctx context.Context, p *ListWorkflowParams) ([]*workflow.Workflow, Cursor, error) {
	got, cursor, err := m.d.Dispatch(ctx, dispatchMethodName(m, "listworkflows"), p)
	if res, ok := got.([]*workflow.Workflow); ok {
		return res, cursor, err
	}
	return nil, cursor, dispatchReturnError(got, err)
}

type GetWorkflowParams struct {
	ID  string
	Ref string
}

func (m AutomationMethods) GetWorkflow(ctx context.Context, p *GetWorkflowParams) (*workflow.Workflow, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "getworkflow"), p)
	if res, ok := got.(*workflow.Workflow); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

type DeployParams = scheduler.DeployParams
type DeployResponse = scheduler.DeployResponse

func (m AutomationMethods) DeployWorkflow(ctx context.Context, p *DeployParams) (*DeployResponse, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "deployworkflow"), p)
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

	// allocate an ID for the transform, for now just log the events it produces
	runID := run.NewID()
	scope.Bus().SubscribeID(func(ctx context.Context, e event.Event) error {
		go func() {
			log.Debugw("apply transform event", "type", e.Type, "payload", e.Payload)
			if e.Type == event.ETTransformPrint {
				if msg, ok := e.Payload.(event.TransformMessage); ok {
					if p.ScriptOutput != nil {
						io.WriteString(p.ScriptOutput, msg.Msg)
						io.WriteString(p.ScriptOutput, "\n")
					}
				}
			}
		}()
		return nil
	}, runID)

	scriptOut := p.ScriptOutput
	transformer := transform.NewTransformer(scope.AppContext(), scope.Loader(), scope.Bus())
	if err = transformer.Apply(ctx, ds, runID, p.Wait, scriptOut, p.Secrets); err != nil {
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

//   automation.workflow          // get a workflow
//   automation.saveWorkflow      // "deploy" in qrimatic UI, create/update a workflow
//   automation.removeWorkflow    // "undeploy" in qrimatic UI

func (automationImpl) ListWorkflows(scp scope, p *ListWorkflowParams) ([]*workflow.Workflow, error) {
	return scp.Workflows().ListWorkflows(scp.Context(), p.Offset, p.Limit)
}

func (automationImpl) GetWorkflow(scp scope, p *GetWorkflowParams) (*workflow.Workflow, error) {
	if p.Ref != "" {
		return scp.Workflows().GetWorkflowByDatasetID(scp.Context(), p.Ref)
	}
	return scp.Workflows().GetWorkflow(scp.Context(), p.ID)
}

// func (automationImpl) SaveWorkflow(scp scope, p *SaveWorkflowParams) (*workflow.Workflow, error) {

// }

// type RemoveWorkflowParams struct {

// }

// func (automationImpl) RemoveWorkflow(scp scope, p *RemoveWorkflowParams) (*workflow.Workflow, error) {

// }

func (automationImpl) DeployWorkflow(scp scope, p *DeployParams) (*DeployResponse, error) {
	if p.Workflow == nil {
		return nil, fmt.Errorf("deploy: workflow not set")
	}
	if p.Workflow.DatasetID == "" {
		return nil, fmt.Errorf("deploy: DatasetID not set")
	}

	wf := p.Workflow
	bus := scp.Bus()
	ctx := scp.Context()

	newWorkflow := true
	if _, err := scp.Workflows().GetWorkflow(ctx, wf.ID); err == nil {
		newWorkflow = false
	}
	if wf.ID == "" && wf.OwnerID != "" && wf.DatasetID != "" {
		wf.ID = workflow.GenerateWorkflowID()
	}

	// TODO(b5): this should be refactored away,
	nowFunc := time.Now

	now := nowFunc()
	if newWorkflow {
		wf.Created = &now
	}
	wf.LatestStart = &now

	go func() {
		if err := bus.PublishID(ctx, workflow.ETWorkflowDeployStarted, wf.ID, wf.Info()); err != nil {
			log.Debugw("async event error", "evt", workflow.ETWorkflowDeployStarted, "workflowID", wf.ID, "err", err)
		}
	}()
	defer func() {
		go func() {
			if err := bus.PublishID(ctx, workflow.ETWorkflowDeployStopped, wf.ID, wf.Info()); err != nil {
				log.Debugw("async event error", "evt", workflow.ETWorkflowDeployStopped, "workflowID", wf.ID, "err", err)
			}
		}()
	}()

	log.Debugw("deploying dataset", "datasetID", wf.DatasetID)

	res, err := datasetImpl{}.Save(scp, &SaveParams{
		Ref: wf.DatasetID, // currently the DatasetID is the Ref
		Dataset: &dataset.Dataset{
			Transform: p.Transform,
		},
		Apply: p.Apply,
		// Wait: false,
	})
	if err != nil {
		if errors.Is(err, dsfs.ErrNoChanges) {
			err = nil
		} else {
			log.Errorw("deploy save dataset", "error", err)
			return nil, err
		}
	}

	now = nowFunc()
	wf.LatestEnd = &now
	wf.RunCount++
	wf.Status = workflow.StatusSucceeded

	if newWorkflow {
		ref := &dsref.Ref{
			Username: res.Peername,
			Name:     res.Name,
		}

		wf.Complete(ref, scp.ActiveProfile().ID.String())
	}

	err = scp.Scheduler().Schedule(ctx, wf)
	if err != nil {
		log.Errorw("deploy scheduling", "error", err)
	}

	return &DeployResponse{
		Workflow: wf,
	}, err
}

// newInstanceRunnerFactory returns a factory function that produces a workflow
// runner from a qri instance
func newInstanceRunnerFactory(inst *Instance) func(ctx context.Context) scheduler.RunWorkflowFunc {
	return func(ctx context.Context) scheduler.RunWorkflowFunc {
		return func(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow) error {
			runID := run.NewID()
			p := &SaveParams{
				Ref: w.DatasetID,
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						RunID: runID,
					},
				},
				Apply: true,
			}
			_, err := inst.Dataset().Save(ctx, p)
			return err
		}
	}
}
