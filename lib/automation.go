package lib

import (
	"context"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/preview"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/scheduler"
	"github.com/qri-io/qri/automation/transform"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
)

// AutomationMethods groups together methods for automations
// TODO(b5): expand apply methods:
//   automation.apply             // Done!
//   automation.workflows         // list local workflows
//   automation.workflow          // get a workflow
//   automation.saveWorkflow      // "deploy" in qrimatic UI, create/update a workflow
//   automation.removeWorkflow    // "undeploy" in qrimatic UI
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
		"apply": {AEApply, "POST"},
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
