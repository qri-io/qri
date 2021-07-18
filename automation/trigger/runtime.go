package trigger

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/qri-io/qri/event"
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
const RuntimeType = "runtime"

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
	bus          event.Bus
	TriggerCh    chan event.WorkflowTriggerEvent
	listening    bool
	triggerStore *MemTriggerStore
}

var _ Listener = (*RuntimeListener)(nil)

// NewRuntimeListener creates a RuntimeListener, and begin receiving on the
// trigger channel. Any triggers received before the RuntimeListener has been
// started using `runtimeListener.Start(ctx)` will be ignored
func NewRuntimeListener(ctx context.Context, bus event.Bus) *RuntimeListener {
	rl := &RuntimeListener{
		bus:       bus,
		TriggerCh: make(chan event.WorkflowTriggerEvent),
	}
	rl.triggerStore = NewMemTriggerStore(rl)
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
	return l.triggerStore.Put(sources...)
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

				err := l.bus.Publish(ctx, event.ETAutomationWorkflowTrigger, wtp)
				if err != nil {
					log.Debugf("RuntimeListener error publishing event.ETAutomationWorkflowTrigger: %s", err)
					continue
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (l *RuntimeListener) shouldTrigger(ctx context.Context, wtp event.WorkflowTriggerEvent) error {
	activeTriggers := l.triggerStore.Active()
	workflowIDs, ok := activeTriggers[wtp.OwnerID]
	if !ok {
		return ErrNotFound
	}
	triggers, ok := workflowIDs[wtp.WorkflowID]
	if !ok {
		return ErrNotFound
	}
	for _, t := range triggers {
		if t.ID() == wtp.TriggerID {
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
	return l.triggerStore.Exists(source)
}
