package automation

import (
	"context"
	"errors"
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
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/base/params"
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
	runLock sync.Mutex
	// TODO(ramfox): `cancelRunCh` is the current shim to ensure we can
	// cancel the currently running run. Like the above `runLock`, this should be
	// replaced with a subsystem that queues/orders/cancels runs
	cancelRunCh  chan string
	workflows    workflow.Store
	listeners    map[string]trigger.Listener
	runs         run.Store
	runFactory   RunFactory
	applyFactory ApplyFactory
	bus          event.Bus
	cancel       context.CancelFunc
	doneCh       chan struct{}
	running      bool
}

// NewOrchestrator constructs an orchestrator
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
			o.Stop()
		}
	}()

	o = &Orchestrator{
		cancel: cancel,
		doneCh: make(chan struct{}),

		bus:          bus,
		runFactory:   runFactory,
		applyFactory: applyFactory,
		workflows:    opts.WorkflowStore,
		runs:         opts.RunStore,
		runLock:      sync.Mutex{},
		cancelRunCh:  make(chan string),
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

		o.listeners = map[string]trigger.Listener{}
	}
	// TODO (ramfox): once hooks/completors are implemented, start the completor system here
	ok = true

	go o.handleContextClose(ctx)

	return o, nil
}

// DefaultOrchestratorOptions is a temporary solution to supplying options to the orchestrator
// TODO (ramfox): remove this in favor of using the automation configuration to
// determing what the orchestrator should be configured as
func DefaultOrchestratorOptions(bus event.Bus, repoPath string) (OrchestratorOptions, error) {
	wfs, err := workflow.NewFileStore(repoPath)
	if err != nil {
		return OrchestratorOptions{}, err
	}
	rs, err := run.NewFileStore(repoPath)
	if err != nil {
		return OrchestratorOptions{}, err
	}
	return OrchestratorOptions{
		WorkflowStore: wfs,
		RunStore:      rs,
		Listeners: []trigger.Listener{
			trigger.NewCronListener(bus),
		},
	}, nil
}

// DefaultMemOrchestratorOptions is primarily for use in tests
// it returns options for an orchestrator whose Stores are in memory implementations
func DefaultMemOrchestratorOptions(ctx context.Context, bus event.Bus) OrchestratorOptions {
	return OrchestratorOptions{
		WorkflowStore: workflow.NewMemStore(),
		RunStore:      run.NewMemStore(),
		Listeners: []trigger.Listener{
			trigger.NewRuntimeListener(ctx, bus),
		},
	}
}

// Start starts the listeners and completors listening for triggers and hooks
func (o *Orchestrator) Start(ctx context.Context) error {
	// TODO(ramfox): when hooks and completors are set up, start them here
	o.running = true
	o.bus.SubscribeTypes(o.handleTrigger, event.ETAutomationWorkflowTrigger)
	return o.startListeners(ctx)
}

// Stop stops the listeners and completors from listening for triggers and hooks
func (o *Orchestrator) Stop() {
	o.cancel()
	<-o.doneCh
}

// Done retrns a read only channel that will close when the Orchestrator
// finishes stopping
func (o *Orchestrator) Done() <-chan struct{} {
	return o.doneCh
}

func (o *Orchestrator) handleContextClose(ctx context.Context) {
	<-ctx.Done()
	o.running = false
	if err := o.workflows.Shutdown(ctx); err != nil {
		log.Errorw("workflows.Shutdown", "error", err)
	}
	if err := o.runs.Shutdown(); err != nil {
		log.Errorw("runs.Shutdown", "error", err)
	}
	// TODO (ramfox): when we have added a way to unsubscribe from a bus, this is where we should do it

	// unsubscribe
	o.stopListeners()
	close(o.doneCh)
}

// startListeners passes a list of deployed Workflows to configured trigger
// Listeners
func (o *Orchestrator) startListeners(ctx context.Context) error {
	wfs, err := o.workflows.ListDeployed(ctx, "", params.ListAll)
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

// advanceTrigger may emit log errors
func (o *Orchestrator) advanceTrigger(wf *workflow.Workflow, triggerID string) *workflow.Workflow {
	for i, triggerOpt := range wf.Triggers {
		trigType := triggerOpt["type"].(string)
		listener, ok := o.listeners[trigType]
		if !ok {
			log.Debugw("advanceTrigger: listener not found", "type", trigType)
			return wf
		}
		trig, err := listener.ConstructTrigger(triggerOpt)
		if err != nil {
			log.Debugw("advanceTrigger: error constructing trigger", "error", err)
			return wf
		}
		if trig.ID() == triggerID {
			trig.Advance()
			w := wf.Copy()
			w.Triggers[i] = trig.ToMap()
			return w
		}
	}
	return wf
}

// handleTrigger calls `RunWorkflow` when an `event.ETAutomationWorkflowTrigger` event is fired
// it expects the payload for the `event.ETAutomationWorkflowTrigger` to be a workflow.ID
// represented as a string
func (o *Orchestrator) handleTrigger(ctx context.Context, e event.Event) error {
	if e.Type == event.ETAutomationWorkflowTrigger {
		wtp, ok := e.Payload.(event.WorkflowTriggerEvent)
		if !ok {
			return fmt.Errorf("handleTrigger: expected event.Payload to be an `event.WorkflowTriggerEvent`: %v", e.Payload)
		}
		go func() {
			wf, err := o.GetWorkflow(ctx, workflow.ID(wtp.WorkflowID))
			if err != nil {
				log.Debugw("handleTrigger: error fetching workflow", "id", wtp.WorkflowID, "err", err)
				return
			}
			wf = o.advanceTrigger(wf, wtp.TriggerID)
			wf, err = o.SaveWorkflow(ctx, wf)
			if err != nil {
				log.Debugw("handleTrigger: error saving workflow", "id", wtp.WorkflowID, "err", err)
			}
			runID := run.NewID()
			if err := o.runWorkflow(ctx, wf, runID); err != nil {
				log.Debugw("handleTrigger: error running workflow", "err", err)
			}
		}()
	}
	return nil
}

// RunWorkflow runs the given workflow
func (o *Orchestrator) RunWorkflow(ctx context.Context, wid workflow.ID, runID string) (string, error) {
	if runID == "" {
		runID = run.NewID()
	}
	wf, err := o.GetWorkflow(ctx, workflow.ID(wid))
	if err != nil {
		return "", err
	}
	return runID, o.runWorkflow(ctx, wf, runID)
}

func (o *Orchestrator) listenForCancelation(ctx context.Context, cancel context.CancelFunc, runID string) {
	log.Debugw("listenForCancelation", "runID", runID)
	for {
		select {
		case cancelRunID := <-o.cancelRunCh:
			if runID == cancelRunID {
				cancel()
				o.runLock.Unlock()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (o *Orchestrator) runWorkflow(ctx context.Context, wf *workflow.Workflow, runID string) error {
	o.runLock.Lock()
	defer o.runLock.Unlock()
	wid := wf.ID
	log.Debugw("runWorkflow, workflow", "id", wid)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go o.listenForCancelation(ctx, cancel, runID)

	go func(wf *workflow.Workflow) {
		if err := o.bus.PublishID(ctx, event.ETAutomationWorkflowStarted, wf.ID.String(), event.WorkflowStartedEvent{
			InitID:     wf.InitID,
			OwnerID:    wf.OwnerID,
			WorkflowID: wf.WorkflowID(),
			RunID:      runID,
		}); err != nil {
			log.Debug(err)
		}
	}(wf)

	if o.runs != nil {
		r := &run.State{ID: runID, WorkflowID: wid}
		if _, err := o.runs.Create(ctx, r); err != nil {
			return err
		}

		handler := runEventsHandler(o.runs)
		o.bus.SubscribeID(handler, runID)
		// TODO (b5): event bus needs an unsubscribe mechanism
		// defer o.bus.UnsubscribeID(runID)
	}

	runFunc := o.runFactory(ctx)
	// need to replace w/ log collector
	streams := ioes.NewDiscardIOStreams()

	err := runFunc(ctx, streams, wf, runID)
	go func(wf *workflow.Workflow) {
		runStatus := run.RSFailed
		if err == nil {
			runStatus = run.RSSucceeded
		}
		if errors.Is(err, dsfs.ErrNoChanges) {
			runStatus = run.RSUnchanged
		}
		if err := o.bus.PublishID(ctx, event.ETAutomationWorkflowStopped, wf.ID.String(), event.WorkflowStoppedEvent{
			InitID:     wf.InitID,
			OwnerID:    wf.OwnerID,
			WorkflowID: wf.WorkflowID(),
			RunID:      runID,
			Status:     string(runStatus),
		}); err != nil {
			log.Debug(err)
		}
	}(wf)

	// TODO (ramfox): when hooks/completors are added, this should wait for the err, iterate through the hooks
	// for this workflow, and emit the events for hooks that this orchestrator understands
	return err
}

// ApplyWorkflow runs the given workflow, but does not record the output
func (o *Orchestrator) ApplyWorkflow(ctx context.Context, wait bool, scriptOutput io.Writer, wf *workflow.Workflow, ds *dataset.Dataset, secrets map[string]string) (string, error) {
	runID := run.NewID()
	if wait {
		return runID, o.applyWorkflow(ctx, scriptOutput, wf, ds, secrets, runID)
	}

	go o.applyWorkflow(ctx, scriptOutput, wf, ds, secrets, runID)
	return runID, nil
}

func (o *Orchestrator) applyWorkflow(ctx context.Context, scriptOutput io.Writer, wf *workflow.Workflow, ds *dataset.Dataset, secrets map[string]string, runID string) error {
	o.runLock.Lock()
	defer o.runLock.Unlock()

	apply := o.applyFactory(ctx)
	log.Debugw("ApplyWorkflow", "workflow id", wf.ID, "run id", runID)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go o.listenForCancelation(ctx, cancel, runID)

	if scriptOutput != nil {
		o.bus.SubscribeID(func(ctx context.Context, e event.Event) error {
			log.Debugw("apply transform event", "type", e.Type, "payload", e.Payload)
			if e.Type == event.ETTransformPrint {
				if msg, ok := e.Payload.(event.TransformMessage); ok {
					if scriptOutput != nil {
						io.WriteString(scriptOutput, msg.Msg)
						io.WriteString(scriptOutput, "\n")
					}
				}
			}
			return nil
		}, runID)
		// TODO (ramfox): defer unsubscribe to id
	}

	// TODO (ramfox): when we understand what it means to dryrun a hook, this should wait for the err, iterator thought the hooks
	// for this workflow, and emit the events for hooks that this orchestrator understands
	return apply(ctx, true, runID, wf, ds, secrets)
}

// CancelRun cancels the run of the given runID
func (o *Orchestrator) CancelRun(ctx context.Context, runID string) {
	log.Debugw("orchestrator.CancelRun", "runID", runID)
	o.cancelRunCh <- runID
}

// SaveWorkflow creates a new workflow if the workflow id is empty, or updates
// an existing workflow in the workflow Store
func (o *Orchestrator) SaveWorkflow(ctx context.Context, wf *workflow.Workflow) (*workflow.Workflow, error) {
	if wf.ID != "" {
		fetchedWF, err := o.workflows.Get(ctx, wf.ID)
		if errors.Is(err, workflow.ErrNotFound) {
			return nil, fmt.Errorf("SaveWorkflow error: workflow %q, %w", wf.ID, err)
		}
		log.Debugw("updating workflow", "new", wf, "old", fetchedWF)
		if fetchedWF.InitID != wf.InitID {
			return nil, fmt.Errorf("SaveWorkflow error: given workflow %q has a different InitID than the workflow on record", wf.ID)
		}
		if fetchedWF.OwnerID != wf.OwnerID {
			return nil, fmt.Errorf("SaveWorkflow error: given workflow %q has a different OwnerID than the workflow on record", wf.ID)
		}
		if wf.Created == nil || !fetchedWF.Created.Equal(*wf.Created) {
			return nil, fmt.Errorf("SaveWorkflow error: given workflow %q has a different Created time than the workflow on record", wf.ID)
		}
	}
	triggers := []map[string]interface{}{}
	for _, opt := range wf.Triggers {
		triggerType, ok := opt["type"].(string)
		if !ok {
			return nil, fmt.Errorf("SaveWorkflow error: trigger options map must include a %q field with the trigger type given as a string", "type")
		}
		listener, ok := o.listeners[triggerType]
		if !ok {
			return nil, fmt.Errorf("SaveWorkflow error: unknown trigger type: %q", triggerType)
		}
		t, err := listener.ConstructTrigger(opt)
		if err != nil {
			return nil, fmt.Errorf("SaveWorkflow error: constructing trigger: %w", err)
		}
		triggers = append(triggers, t.ToMap())
	}
	wf.Triggers = triggers
	// TODO (ramfox): when we add hooks in a follow up, this function should receive HookrOptions as a param

	// it should iterate over the hooks this orchestrator understands, and err if the workflow references
	// any that it doesn't know about

	isNewWF := wf.ID == ""
	if isNewWF {
		wf.Created = NowFunc()
	}
	wf, err := o.workflows.Put(ctx, wf)
	if err != nil {
		return nil, err
	}
	if isNewWF {
		go func() {
			if err := o.bus.Publish(ctx, event.ETAutomationWorkflowCreated, *wf); err != nil {
				log.Debug(err)
			}
		}()
	}
	go o.updateListeners(wf)
	return wf, err
}

// GetWorkflow fetches an existing workflow from the WorkflowStore
func (o *Orchestrator) GetWorkflow(ctx context.Context, id workflow.ID) (*workflow.Workflow, error) {
	return o.workflows.Get(ctx, id)
}

// GetWorkflowByInitID fetches an existing workflow from the WorkflowStore by the InitID
func (o *Orchestrator) GetWorkflowByInitID(ctx context.Context, id string) (*workflow.Workflow, error) {
	return o.workflows.GetByInitID(ctx, id)
}

// RemoveWorkflow removes a workflow form the workflow.Store
func (o *Orchestrator) RemoveWorkflow(ctx context.Context, id workflow.ID) error {
	wf, err := o.workflows.Get(ctx, id)
	if err != nil {
		return err
	}
	wf.Triggers = []map[string]interface{}{}
	if err := o.workflows.Remove(ctx, id); err != nil {
		return err
	}
	go func() {
		if err := o.bus.Publish(ctx, event.ETAutomationWorkflowRemoved, *wf); err != nil {
			log.Debug(err)
		}
	}()
	go o.updateListeners(wf)
	return nil
}

// runEventsHandler returns a handler that writes run events to a run store
func runEventsHandler(store run.Store) event.Handler {
	return func(ctx context.Context, e event.Event) error {
		if adder, ok := store.(run.EventAdder); ok {
			return adder.AddEvent(e.SessionID, e)
		}

		r, err := store.Get(ctx, e.SessionID)
		if err != nil {
			return err
		}
		if err := r.AddTransformEvent(e); err != nil {
			return err
		}

		_, err = store.Put(ctx, r)
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
