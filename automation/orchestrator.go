package automation

import (
	"context"
	"fmt"
	"sync"
	"time"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
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
	WorkflowStore workflow.Store
	RunStore      run.Store
}

// Run persists the dataset that results from executing a workflow transform
type Run func(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow, runID string) error

// RunFactory is a function that produces a Run function
type RunFactory func(ctx context.Context) Run

// Apply executes an ephemeral workflow transform
type Apply func(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow) error

// ApplyFactory is function that produces an Apply function
type ApplyFactory func(ctx context.Context) Apply

// Orchestrator manages automation in qri
type Orchestrator struct {
	// TODO(ramfox): this runLock is the current shim to ensure only one workflow runs at a time
	// we should probably have a run queue subsystem that ensure the orchestrator is running
	// the workflows in the expected order, running only as many at once as configured, and
	// allows communication back to the user about where they are in the run queue, allows for
	// cancelling runs that haven't happened yet
	runLock      sync.Mutex
	workflows    workflow.Store
	runs         run.Store
	runFactory   RunFactory
	applyFactory ApplyFactory
	bus          event.Bus
	cancel       context.CancelFunc
}

// NewOrchestrator constructs an orchestrator, whose only responsibility, right
// now, is to create a workflow store, run store, & listen for trigger events
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
		runs:         opts.RunStore,
	}

	if o.workflows == nil {
		// TODO(ramfox): once we have a `config.Automation` specified, we will have a
		// specific `workflow.NewStore` function that takes a `config.Workflow` & will
		// return a specified `workflow.Store`
		return nil, fmt.Errorf("no workflow store specified")
	}

	if o.runs == nil {
		// TODO(ramfox): once we have a `config.Automation` specified, we will have a
		// specific `run.NewStore` function that takes a `config.RunStore` & will
		// return a specified `run.Store`
		return nil, fmt.Errorf("no run store specified")
	}

	o.bus.SubscribeTypes(o.handleTrigger, event.ETWorkflowTrigger)
	// TODO (ramfox): once hooks/completors are implemented, start the completor system here
	ok = true
	return o, nil
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
		wid, ok := e.Payload.(workflow.ID)
		if !ok {
			return fmt.Errorf("handleTrigger: expected event.Payload to be a workflow.ID: %v", e.Payload)
		}
		go func() {
			if err := o.RunWorkflow(ctx, wid); err != nil {
				log.Errorf("%s", err)
			}
		}()
	}
	return nil
}

// RunWorkflow runs the given workflow
func (o *Orchestrator) RunWorkflow(ctx context.Context, wid workflow.ID) error {
	o.runLock.Lock()
	defer o.runLock.Unlock()
	log.Debugw("RunWorkflow, workflow", "id", wid)
	runFunc := o.runFactory(ctx)
	wf, err := o.GetWorkflow(wid)
	if err != nil {
		log.Debugw("RunWorkflow: getting workflow from store", "err", err)
		return fmt.Errorf("getting workflow from store: %w", err)
	}
	// need to replace w/ log collector
	streams := ioes.NewDiscardIOStreams()

	// TODO (ramfox): when hooks/completors are added, this should wait for the err, iterate through the hooks
	// for this workflow, and emit the events for hooks that this orchestrator understands
	runID := run.NewID()
	if o.runs != nil {
		r := &run.State{ID: runID, WorkflowID: wid}
		if _, err := o.runs.Create(r); err != nil {
			return err
		}

		handler := runEventsHandler(o.runs)
		o.bus.SubscribeID(handler, runID)
		// TODO (b5): event bus needs an unsubscribe mechanism
		// defer o.bus.UnsubscribeID(runID)
	}

	return runFunc(ctx, streams, wf, runID)
}

// ApplyWorkflow runs the given workflow, but does not record the output
func (o *Orchestrator) ApplyWorkflow(ctx context.Context, wid workflow.ID) error {
	o.runLock.Lock()
	defer o.runLock.Unlock()
	log.Debugw("ApplyWorkflow, workflow", "id", wid)
	apply := o.applyFactory(ctx)
	wf, err := o.GetWorkflow(wid)
	if err != nil {
		log.Debugw("ApplyWorkflow: getting workflow from store", "err", err)
		return fmt.Errorf("getting workflow from store: %w", err)
	}
	streams := ioes.NewDiscardIOStreams()

	// TODO (ramfox): when we understand what it means to dryrun a hook, this should wait for the err, iterator thought the hooks
	// for this workflow, and emit the events for hooks that this orchestrator understands
	return apply(ctx, streams, wf)
}

// CreateWorkflow creates a new workflow and adds it to the WorkflowStore
func (o *Orchestrator) CreateWorkflow(did string, pid profile.ID) (*workflow.Workflow, error) {
	// TODO (ramfox): when we add triggers in a follow up, this function should receive TriggerOptions as a param
	// it should iterate over the triggers this orchestrator understands, and err if the workflow references
	// any that it doesn't know about
	// it should convert each TriggerOption into a Trigger & pass them down to `workflow.Create`
	// TODO (ramfox): same goes for HookOptions & hooks
	wf := &workflow.Workflow{
		DatasetID: did,
		OwnerID:   pid,
		Created:   NowFunc(),
	}
	return o.workflows.Put(wf)
}

// GetWorkflow fetches an existing workflow from the WorkflowStore
func (o *Orchestrator) GetWorkflow(id workflow.ID) (*workflow.Workflow, error) {
	return o.workflows.Get(id)
}

// runEventsHandler returns a handler that writes run events to a run store
func runEventsHandler(store run.Store) event.Handler {
	return func(ctx context.Context, e event.Event) error {
		if adder, ok := store.(run.EventAdder); ok {
			return adder.AddEvent(e.SessionID, e)
		}

		r, err := store.Get(e.SessionID)
		if err != nil {
			return err
		}
		if err := r.AddTransformEvent(e); err != nil {
			return err
		}

		_, err = store.Put(r)
		return err
	}
}
