package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
)

// A RuntimeTrigger implements the Trigger interface & keeps track of the number
// of times it had been advanced
type RuntimeTrigger struct {
	id           string
	active       bool
	AdvanceCount int
}

var _ Trigger = (*RuntimeTrigger)(nil)

// RuntimeType denotes a `RuntimeTrigger`
const RuntimeType = "Runtime Trigger"

// NewRuntimeTrigger returns an active `RuntimeTrigger`
func NewRuntimeTrigger() *RuntimeTrigger {
	return &RuntimeTrigger{
		id:           NewID(),
		active:       false,
		AdvanceCount: 0,
	}
}

// ID return the trigger.ID
func (rt *RuntimeTrigger) ID() string {
	return rt.id
}

// Active returns if the RuntimeTrigger is active
func (rt *RuntimeTrigger) Active() bool {
	return rt.active
}

// SetActive sets the active status
func (rt *RuntimeTrigger) SetActive(active bool) error {
	rt.active = active
	return nil
}

// Type returns the RuntimeType
func (rt *RuntimeTrigger) Type() string {
	return RuntimeType
}

// Advance increments the AdvanceCount
func (rt *RuntimeTrigger) Advance() error {
	rt.AdvanceCount++
	return nil
}

// ToMap returns the trigger as a map[string]interface{}
func (rt *RuntimeTrigger) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":           rt.id,
		"active":       rt.active,
		"type":         RuntimeType,
		"advanceCount": rt.AdvanceCount,
	}
}

type runtimeTrigger struct {
	ID           string `json:"id"`
	Active       bool   `json:"active"`
	Type         string `json:"type"`
	AdvanceCount int    `json:"advanceCount"`
}

// MarshalJSON implements the json.Marshaller interface
func (rt *RuntimeTrigger) MarshalJSON() ([]byte, error) {
	if rt == nil {
		rt = &RuntimeTrigger{}
	}
	return json.Marshal(runtimeTrigger{
		ID:           rt.ID(),
		Active:       rt.active,
		Type:         rt.Type(),
		AdvanceCount: rt.AdvanceCount,
	})
}

// UnmarshalJSON implements the json.Unmarshaller interface
func (rt *RuntimeTrigger) UnmarshalJSON(d []byte) error {
	t := &runtimeTrigger{}
	err := json.Unmarshal(d, t)
	if err != nil {
		return err
	}
	if t.Type != RuntimeType {
		return fmt.Errorf("%w, got %s, expected %s", ErrUnexpectedType, t.Type, RuntimeType)
	}
	*rt = RuntimeTrigger{
		id:           t.ID,
		active:       t.Active,
		AdvanceCount: t.AdvanceCount,
	}
	return nil
}

// RuntimeListener listens for RuntimeTriggers to fire
type RuntimeListener struct {
	bus                event.Bus
	TriggerCh          chan *event.WorkflowTriggerPayload
	listening          bool
	activeTriggersLock sync.Mutex
	activeTriggers     map[profile.ID]map[string][]string
}

// NewRuntimeListener creates a RuntimeListener, and begin receiving on the
// trigger channel. Any triggers received before the RuntimeListener has been
// started using `runtimeListener.Start(ctx)` will be ignored
func NewRuntimeListener(ctx context.Context, bus event.Bus) *RuntimeListener {
	rl := &RuntimeListener{
		bus:            bus,
		TriggerCh:      make(chan *event.WorkflowTriggerPayload),
		activeTriggers: map[profile.ID]map[string][]string{},
	}
	// start ensures that if a RuntimeTrigger attempts to trigger a workflow,
	// but the RuntimeListener has not been told to start listening for
	// triggers, the RuntimeTrigger won't block waiting for the
	// RuntimeListener to start. Instead, the trigger will just get ignored
	go rl.start(ctx)
	return rl
}

// ConstructTrigger creates a RuntimeTrigger from a map string interface config
// The map must have a field "type" of type RuntimeTrigger
func (l *RuntimeListener) ConstructTrigger(opt map[string]interface{}) (Trigger, error) {
	t := opt["type"]
	if t != l.Type() {
		return nil, fmt.Errorf("%w, expected %q but got %q", ErrTypeMismatch, l.Type(), t)
	}

	data, err := json.Marshal(opt)
	if err != nil {
		return nil, err
	}
	rt := &RuntimeTrigger{}
	err = rt.UnmarshalJSON(data)

	if rt.id == "" {
		rt.id = NewID()
	}
	return rt, err
}

// Listen takes a list of sources and adds or updates the Listener's
// store to include all the active triggers of the correct type
func (l *RuntimeListener) Listen(sources ...Source) error {
	l.activeTriggersLock.Lock()
	defer l.activeTriggersLock.Unlock()

	for _, s := range sources {
		workflowID := s.WorkflowID()
		if workflowID == "" {
			return ErrEmptyWorkflowID
		}
		ownerID := s.Owner()
		if ownerID == "" {
			return ErrEmptyOwnerID
		}
		triggerOpts := s.ActiveTriggers(RuntimeType)
		triggers := []Trigger{}
		for _, triggerOpt := range triggerOpts {
			t, err := l.ConstructTrigger(triggerOpt)
			if err != nil {
				return err
			}
			triggers = append(triggers, t)
		}
		wids, ok := l.activeTriggers[ownerID]
		if !ok {
			if len(triggers) == 0 {
				continue
			}
			l.activeTriggers[ownerID] = map[string][]string{}
			wids = l.activeTriggers[ownerID]
		}
		tids, ok := wids[workflowID]
		if !ok {
			if len(triggers) == 0 {
				continue
			}
			l.activeTriggers[ownerID][workflowID] = []string{}
			tids = l.activeTriggers[ownerID][workflowID]
		}
		if len(triggers) == 0 {
			delete(l.activeTriggers[ownerID], workflowID)
			continue
		}
		tids = []string{}
		for _, t := range triggers {
			tids = append(tids, t.ID())
		}
		l.activeTriggers[ownerID][workflowID] = tids
	}
	return nil
}

// Type returns the Type `RuntimeType`
func (l *RuntimeListener) Type() string {
	return RuntimeType
}

func (l *RuntimeListener) start(ctx context.Context) error {
	go func() {
		for {
			select {
			case wtp := <-l.TriggerCh:
				if !l.listening {
					log.Debugf("RuntimeListener: trigger ignored")
					continue
				}
				if err := l.shouldTrigger(ctx, wtp); err != nil {
					log.Debugf("RuntimeListener error: %s", err)
					continue
				}

				err := l.bus.Publish(ctx, event.ETWorkflowTrigger, wtp)
				if err != nil {
					log.Debugf("RuntimeListener error publishing event.ETWorkflowTrigger: %s", err)
					continue
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (l *RuntimeListener) shouldTrigger(ctx context.Context, wtp *event.WorkflowTriggerPayload) error {
	l.activeTriggersLock.Lock()
	defer l.activeTriggersLock.Unlock()

	workflowIDs, ok := l.activeTriggers[wtp.OwnerID]
	if !ok {
		return ErrNotFound
	}
	triggerIDs, ok := workflowIDs[wtp.WorkflowID]
	if !ok {
		return ErrNotFound
	}
	for _, tid := range triggerIDs {
		if tid == wtp.TriggerID {
			return nil
		}
	}
	return ErrNotFound
}

// Start tells the RuntimeListener to begin actively listening for RuntimeTriggers
func (l *RuntimeListener) Start(ctx context.Context) error {
	l.listening = true
	go func() {
		select {
		case <-ctx.Done():
			l.Stop()
		}
	}()
	return nil
}

// Stop tells the RuntimeListener to stop actively listening for RuntimeTriggers
func (l *RuntimeListener) Stop() error {
	l.listening = false
	return nil
}

// TriggersExists returns true if triggers in the source match the triggers stored in
// the runtime listener
func (l *RuntimeListener) TriggersExists(source Source) bool {
	l.activeTriggersLock.Lock()
	defer l.activeTriggersLock.Unlock()

	ownerID := source.Owner()
	workflowID := source.WorkflowID()
	wids, ok := l.activeTriggers[ownerID]
	if !ok {
		return false
	}
	tids, ok := wids[workflowID]
	if !ok {
		return false
	}
	triggerOpts := source.ActiveTriggers(RuntimeType)
	if len(triggerOpts) != len(tids) {
		return false
	}
	for i, opt := range triggerOpts {
		t, err := l.ConstructTrigger(opt)
		if err != nil {
			log.Errorw("runtimeListener.TriggersExist", "error", err)
			return false
		}
		if t.ID() != tids[i] {
			return false
		}
	}
	return true
}
