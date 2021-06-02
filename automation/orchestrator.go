package automation

import (
	"context"
	"fmt"
	"io"
	"time"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/trigger"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/event"
)

var (
	log = golog.Logger("automation")
)

// NowFunc returns a pointer to the current time. Can be overridden in
// tests to create determinism
var NowFunc = func() *time.Time {
	now := time.Now()
	return &now
}

// OrchestratorOptions encapsulate runtime configuration for NewOrchestrator
type OrchestratorOptions struct {
	WorkflowStore    workflow.Store
	TriggerListeners []TriggerListener
}

// Listener emits a `event.ETTriggerWorkflow` when a specific stimulus is triggered
// It knows how to start and stop itself, as well as how to create new triggers for its specific stimulus
type TriggerListener interface {
	Listen(sources ...trigger.Source)
	Start(ctx context.Context) error
	Stop() error
}

// Run persists the dataset that results from executing a workflow transform
type Run func(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow, runID string) error

// RunFactory is a function that produces a Run function
type RunFactory func(ctx context.Context) Run

// Apply executes an ephemeral workflow transform
type Apply func(ctx context.Context, wait bool, runID string, w *workflow.Workflow, ds *dataset.Dataset, secrets map[string]string) error

// ApplyFactory is function that produces an Apply function
type ApplyFactory func(ctx context.Context) Apply

// Orchestrator manages automation in qri
type Orchestrator struct {
	workflows    workflow.Store
	runFactory   RunFactory
	applyFactory ApplyFactory
	bus          event.Bus
	cancel       context.CancelFunc
	listeners    []TriggerListener
}

// NewOrchestrator constructs an orchestrator, whose only responsibility,
// right now, is to create a workflow store & listen for trigger events
func NewOrchestrator(ctx context.Context, bus event.Bus, runFactory RunFactory, applyFactory ApplyFactory, opts OrchestratorOptions) (*Orchestrator, error) {
	log.Debugw("NewOrchestrator", "opts", opts)

	if bus == nil {
		return nil, fmt.Errorf("bus of type event.Bus required")
	}
	if runFactory == nil {
		return nil, fmt.Errorf("runFactory of type RunFactory required")
	}
	if applyFactory == nil {
		return nil, fmt.Errorf("applyFactory of type ApplyFactory required")
	}

	ctx, cancel := context.WithCancel(ctx)
	ok := false
	var o *Orchestrator
	defer func() {
		if !ok {
			o.Shutdown()
		}
	}()

	o = &Orchestrator{
		cancel:       cancel,
		bus:          bus,
		runFactory:   runFactory,
		applyFactory: applyFactory,
		workflows:    opts.WorkflowStore,
		listeners:    opts.TriggerListeners,
	}

	if o.workflows == nil {
		// TODO(ramfox): once we have a `config.Automation` specified, we will have a
		// specific `workflow.NewStore` function that takes a `config.Workflow` & will
		// return a specified `workflow.Store`
		return nil, fmt.Errorf("no workflow store specified")
	}

	o.bus.SubscribeTypes(o.handleTrigger, event.ETWorkflowTrigger)
	// TODO (ramfox): once hooks/completors are implemented, start the completor system here
	ok = true
	return o, nil
}

func (o *Orchestrator) Start(ctx context.Context) error {
	// iterate over listeners & call listener.Start(ctx)
	wfs, err := o.workflows.List(ctx, 0, -1)
	if err != nil {
		return err
	}

	srcs := make([]trigger.Source, 0, len(wfs))
	for _, wf := range wfs {
		srcs = append(srcs, wf)
	}

	for _, l := range o.listeners {
		l.Listen(srcs...)
		go l.Start(ctx)
	}

	return nil
}

// Shutdown tears down the orchestrator system
// must be called to ensure all processes have be closed correctly
func (o *Orchestrator) Shutdown() {
	// TODO (ramfox): when we have added a way to unsubscribe from a bus, this is where we should do it
	o.cancel()
}

// handleTrigger calls `RunWorkflow` when an `event.ETWorkflowTrigger` event is fired
func (o *Orchestrator) handleTrigger(ctx context.Context, e event.Event) error {
	if e.Type == event.ETWorkflowTrigger {
		var wid workflow.ID
		switch pl := e.Payload.(type) {
		case string:
			wid = workflow.ID(pl)
		case workflow.ID:
			wid = pl
		default:
			return fmt.Errorf("handleTrigger: expected event.Payload to be a workflow.ID: %v", e.Payload)
		}

		_, err := o.RunWorkflow(ctx, wid)
		return err
	}
	return nil
}

// RunWorkflow runs the given workflow
func (o *Orchestrator) RunWorkflow(ctx context.Context, wid workflow.ID) (string, error) {
	log.Debugw("RunWorkflow, workflow", "id", wid)
	runFunc := o.runFactory(ctx)

	wf, err := o.workflows.Get(wid)
	if err != nil {
		log.Debugw("RunWorkflow: getting workflow from store", "wid", wid, "err", err)
		return "", fmt.Errorf("getting workflow from store: %w", err)
	}
	// need to replace w/ log collector
	streams := ioes.NewDiscardIOStreams()

	runID := run.NewID()

	go func(wf *workflow.Workflow) {
		if err := o.bus.Publish(ctx, event.ETWorkflowStarted, wf); err != nil {
			log.Debug(err)
		}
	}(wf)

	// TODO (ramfox): when hooks/completors are added, this should wait for the err, iterate through the hooks
	// for this workflow, and emit the events for hooks that this orchestrator understands
	err = runFunc(ctx, streams, wf, runID)

	go func(j *workflow.Workflow) {
		if err := o.bus.Publish(ctx, event.ETWorkflowCompleted, j); err != nil {
			log.Debug(err)
		}
	}(wf)

	return runID, err
}

// ApplyWorkflow runs the given workflow, but does not record the output
func (o *Orchestrator) ApplyWorkflow(ctx context.Context, wait bool, scriptOutput io.Writer, wf *workflow.Workflow, ds *dataset.Dataset, secrets map[string]string) (string, error) {

	apply := o.applyFactory(ctx)

	runID := run.NewID()
	if scriptOutput != nil {
		o.bus.SubscribeID(func(ctx context.Context, e event.Event) error {
			go func() {
				log.Debugw("apply transform event", "type", e.Type, "payload", e.Payload)
				if e.Type == event.ETTransformPrint {
					if msg, ok := e.Payload.(event.TransformMessage); ok {
						if scriptOutput != nil {
							io.WriteString(scriptOutput, msg.Msg)
							io.WriteString(scriptOutput, "\n")
						}
					}
				}
			}()
			return nil
		}, runID)
	}

	// TODO (ramfox): when we understand what it means to dryrun a hook, this should wait for the err, iterator thought the hooks
	// for this workflow, and emit the events for hooks that this orchestrator understands
	return runID, apply(ctx, wait, runID, wf, ds, secrets)
}

// CreateWorkflow creates a new workflow and adds it to the WorkflowStore
func (o *Orchestrator) SaveWorkflow(ctx context.Context, wf *workflow.Workflow) (*workflow.Workflow, error) {
	if wf.ID == "" {
		wf.Created = NowFunc()
	}

	wf, err := o.workflows.Put(wf)
	if err != nil {
		return nil, err
	}

	if wf.Deployed {
		// go func() {
		if err := o.bus.PublishID(ctx, event.ETWorkflowDeployStarted, wf.ID.String(), wf); err != nil {
			log.Debugw("async event error", "evt", event.ETWorkflowDeployStarted, "workflowID", wf.ID, "err", err)
		}
		// }()

		for _, l := range o.listeners {
			l.Listen(wf)
		}

		// go func() {
		if err := o.bus.PublishID(ctx, event.ETWorkflowDeployStopped, wf.ID.String(), wf); err != nil {
			log.Debugw("async event error", "evt", event.ETWorkflowDeployStopped, "workflowID", wf.ID, "err", err)
		}
		// }()
	}

	return wf, err
}

// GetWorkflow fetches an existing workflow from the WorkflowStore
func (o *Orchestrator) GetWorkflow(id workflow.ID) (*workflow.Workflow, error) {
	return o.workflows.Get(id)
}

func (o *Orchestrator) Workflows() workflow.Store {
	return o.workflows
}
