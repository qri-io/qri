package automation

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
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
type Apply func(ctx context.Context, wait bool, runID string, w *workflow.Workflow, ds *dataset.Dataset, secrets map[string]string) error

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
	o.bus.SubscribeTypes(o.handleTrigger, event.ETWorkflowTrigger)
	// TODO (ramfox): once hooks/completors are implemented, start the completor system here
	ok = true
	return o, nil
}

// Start starts the listeners and completors listening for triggers and hooks
func (o *Orchestrator) Start(ctx context.Context) error {
	// TODO(ramfox): when hooks and completors are set up, start them here
	return o.startListeners(ctx)
}

// Stop stops the listeners and completors from listening for triggers and hooks
func (o *Orchestrator) Stop() {
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
		wid, ok := e.Payload.(string)
		if !ok {
			return fmt.Errorf("handleTrigger: expected event.Payload to be a string: %v", e.Payload)
		}
		go func() {
			if _, err := o.RunWorkflow(ctx, workflow.ID(wid)); err != nil {
				log.Errorf("%s", err)
			}
		}()
	}
	return nil
}

// RunWorkflow runs the given workflow
func (o *Orchestrator) RunWorkflow(ctx context.Context, wid workflow.ID) (string, error) {
	o.runLock.Lock()
	defer o.runLock.Unlock()
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
	if o.runs != nil {
		r := &run.State{ID: runID, WorkflowID: wid}
		if _, err := o.runs.Create(r); err != nil {
			return "", err
		}

		handler := runEventsHandler(o.runs)
		o.bus.SubscribeID(handler, runID)
		// TODO (b5): event bus needs an unsubscribe mechanism
		// defer o.bus.UnsubscribeID(runID)
	}

	err = runFunc(ctx, streams, wf, runID)

	go func(wf *workflow.Workflow) {
		if err := o.bus.Publish(ctx, event.ETWorkflowCompleted, wf); err != nil {
			log.Debug(err)
		}
	}(wf)

	return runID, err
}

// ApplyWorkflow runs the given workflow, but does not record the output
// func (o *Orchestrator) ApplyWorkflow(ctx context.Context, wait bool, scriptOutput io.Writer, wf *workflow.Workflow, ds *dataset.Dataset, secrets map[string]string) (string, error) {
func (o *Orchestrator) ApplyWorkflow(ctx context.Context, wait bool, scriptOutput io.Writer, wf *workflow.Workflow, ds *dataset.Dataset, secrets map[string]string) (string, error) {
	o.runLock.Lock()
	defer o.runLock.Unlock()
	log.Debugw("ApplyWorkflow, workflow", "id", wf.ID)
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

// DeployWorkflow deploys a workflow
func (o *Orchestrator) DeployWorkflow(id workflow.ID) error {
	wf, err := o.workflows.Get(id)
	if err != nil {
		return err
	}
	wf.Deployed = true
	_, err = o.workflows.Put(wf)
	o.updateListeners(wf)
	return err
}

// UndeployWorkflow undeploys a workflow
func (o *Orchestrator) UndeployWorkflow(id workflow.ID) error {
	wf, err := o.workflows.Get(id)
	if err != nil {
		return err
	}
	wf.Deployed = false
	_, err = o.workflows.Put(wf)
	o.updateListeners(wf)
	return err
}

func (o *Orchestrator) Workflows() workflow.Store {
	return o.workflows
}

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
	for _, listener := range o.listeners {
		go func(l trigger.Listener) {
			err := l.Listen(sources...)
			if err != nil {
				log.Debugf("error updating triggers for listener %q", l.Type())
			}

		}(listener)
	}
}
