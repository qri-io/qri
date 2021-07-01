package automation

import (
	"context"
	"fmt"
	"sync"
	"time"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/trigger"
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
	Listeners     []trigger.Listener
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
	listeners    map[string]trigger.Listener
	runs         run.Store
	runFactory   RunFactory
	applyFactory ApplyFactory
	bus          event.Bus
	cancel       context.CancelFunc
	running      bool
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

	for _, l := range opts.Listeners {
		if o.listeners == nil {
			o.listeners = map[string]trigger.Listener{}
		}
		if _, ok := o.listeners[l.Type()]; ok {
			return nil, fmt.Errorf("multiple trigger listeners of type %q specified - can only have one of each type of listener", l.Type())
		}
		o.listeners[l.Type()] = l
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

	if o.listeners == nil {
		// TODO(ramfox): once we have a `config.Automation` specified, we will have a
		// specific `trigger.NewListeners` function that takes a `config.Listeners` & will
		// return a list of specified listeners
		// Need to decide if a user can use a combination of the list of options given by
		// the config & the list of listeners given by the options to define a list of listners
		// that this orchestrator will use, or if it must be one or the other.
		return nil, fmt.Errorf("no listeners specified")
	}
	// TODO (ramfox): once hooks/completors are implemented, start the completor system here
	ok = true
	return o, nil
}

// Start starts the listeners and completors listening for triggers and hooks
func (o *Orchestrator) Start(ctx context.Context) error {
	// TODO(ramfox): when hooks and completors are set up, start them here
	o.running = true
	o.bus.SubscribeTypes(o.handleTrigger, event.ETWorkflowTrigger)
	return o.startListeners(ctx)
}

// Stop stops the listeners and completors from listening for triggers and hooks
func (o *Orchestrator) Stop() {
	// unsubscribe
	o.running = false
	o.stopListeners()
}

// startListeners passes a list of deployed Workflows to configured trigger
// Listeners
func (o *Orchestrator) startListeners(ctx context.Context) error {
	wfs, err := o.workflows.ListDeployed(ctx, -1, 0)
	if err != nil {
		return fmt.Errorf("error getting deployed workflows from the store: %w", err)
	}
	srcs := make([]trigger.Source, 0, len(wfs))
	for _, wf := range wfs {
		srcs = append(srcs, wf)
	}

	for _, listener := range o.listeners {
		go func(l trigger.Listener) {
			err := l.Listen(srcs...)
			if err != nil {
				log.Debug(err)
				return
			}
			err = l.Start(ctx)
			if err != nil {
				log.Debug(err)
			}
		}(listener)
	}
	return nil
}

// stopListeners stops the orchestrator's trigger.Listeners from listening for
// triggers
func (o *Orchestrator) stopListeners() {
	for _, listeners := range o.listeners {
		err := listeners.Stop()
		if err != nil {
			log.Debugf("Orchestrator StopListeners error: %s", err)
		}
	}
}

// Shutdown stops any currently running processes and tears down the orchestrator system
// must be called to ensure all processes have be closed correctly
func (o *Orchestrator) Shutdown() {
	// TODO (ramfox): when we have added a way to unsubscribe from a bus, this is where we should do it
	o.Stop()
	o.cancel()
}

// handleTrigger calls `RunWorkflow` when an `event.ETWorkflowTrigger` event is fired
// it expects the payload for the `event.ETWorkflowTrigger` to be a workflow.ID
// represented as a string
func (o *Orchestrator) handleTrigger(ctx context.Context, e event.Event) error {
	if e.Type == event.ETWorkflowTrigger {
		wtp, ok := e.Payload.(*event.WorkflowTriggerPayload)
		if !ok {
			return fmt.Errorf("handleTrigger: expected event.Payload to be an `event.WorkflowTriggerPayload`: %v", e.Payload)
		}
		go func() {
			wf, err := o.GetWorkflow(workflow.ID(wtp.WorkflowID))
			if err != nil {
				log.Debugw("handleTrigger: error fetching workflow", "id", wtp.WorkflowID, "err", err)
				return
			}
			for _, trig := range wf.Triggers {
				if trig.ID() == wtp.TriggerID {
					trig.Advance()
					break
				}
			}
			wf, err = o.UpdateWorkflow(wf)
			if err != nil {
				log.Debugw("handleTrigger: error saving workflow", "id", wtp.WorkflowID, "err", err)
			}
			if err := o.runWorkflow(ctx, wf); err != nil {
				log.Debugw("handleTrigger: error running workflow", "err", err)
			}
		}()
	}
	return nil
}

// RunWorkflow runs the given workflow
func (o *Orchestrator) RunWorkflow(ctx context.Context, wid workflow.ID) error {
	wf, err := o.GetWorkflow(workflow.ID(wid))
	if err != nil {
		return err
	}
	return o.runWorkflow(ctx, wf)
}

func (o *Orchestrator) runWorkflow(ctx context.Context, wf *workflow.Workflow) error {
	o.runLock.Lock()
	defer o.runLock.Unlock()
	wid := wf.ID
	log.Debugw("runWorkflow, workflow", "id", wid)
	runFunc := o.runFactory(ctx)
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
func (o *Orchestrator) CreateWorkflow(did string, pid profile.ID, triggerOpts []map[string]interface{}) (*workflow.Workflow, error) {
	t := []trigger.Trigger{}
	for _, opt := range triggerOpts {
		triggerType, ok := opt["type"].(string)
		if !ok {
			return nil, fmt.Errorf("trigger options map must include a %q field with the trigger type given as a string", "type")
		}
		listener, ok := o.listeners[triggerType]
		if !ok {
			return nil, fmt.Errorf("CreateWorkflow unknown trigger type: %q", triggerType)
		}
		trig, err := listener.ConstructTrigger(opt)
		if err != nil {
			return nil, fmt.Errorf("CreateWorkflow error constructing trigger: %w", err)
		}
		t = append(t, trig)
	}
	// TODO (ramfox): when we add hooks in a follow up, this function should receive HookrOptions as a param
	// it should iterate over the hooks this orchestrator understands, and err if the workflow references
	// any that it doesn't know about

	// it should convert each HookOption into a Hook & pass them down to `workflow.Create`
	wf := &workflow.Workflow{
		DatasetID: did,
		OwnerID:   pid,
		Created:   NowFunc(),
		Triggers:  t,
	}
	wf, err := o.workflows.Put(wf)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

// UpdateWorkflow updates a workflow in the workflow.Store
func (o *Orchestrator) UpdateWorkflow(wf *workflow.Workflow) (*workflow.Workflow, error) {
	return o.workflows.Put(wf)
}

// DeployWorkflow deploys a workflow
func (o *Orchestrator) DeployWorkflow(id workflow.ID) (*workflow.Workflow, error) {
	wf, err := o.workflows.Get(id)
	if err != nil {
		return nil, err
	}
	wf.Deployed = true
	defer o.updateListeners(wf)
	return o.workflows.Put(wf)
}

// UndeployWorkflow undeploys a workflow
func (o *Orchestrator) UndeployWorkflow(id workflow.ID) (*workflow.Workflow, error) {
	wf, err := o.workflows.Get(id)
	if err != nil {
		return nil, err
	}
	wf.Deployed = false
	defer o.updateListeners(wf)
	return o.workflows.Put(wf)
}

// GetWorkflow fetches an existing workflow from the WorkflowStore
func (o *Orchestrator) GetWorkflow(id workflow.ID) (*workflow.Workflow, error) {
	return o.workflows.Get(id)
}

// RemoveWorkflow removes a workflow form the workflow.Store
func (o *Orchestrator) RemoveWorkflow(id workflow.ID) error {
	return o.workflows.Remove(id)
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

func (o *Orchestrator) updateListeners(sources ...trigger.Source) {
	if !o.running {
		return
	}
	for _, listener := range o.listeners {
		go func(l trigger.Listener) {
			err := l.Listen(sources...)
			if err != nil {
				log.Debugf("error updating triggers for listener %q", l.Type())
			}

		}(listener)
	}
}
