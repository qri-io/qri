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
	active       bool
	AdvanceCount int
}

var _ Trigger = (*RuntimeTrigger)(nil)

// RuntimeType denotes a `RuntimeTrigger`
var RuntimeType = Type("Runtime Trigger")

// NewRuntimeTrigger returns an active `RuntimeTrigger`
func NewRuntimeTrigger() *RuntimeTrigger {
	return &RuntimeTrigger{
		active:       false,
		AdvanceCount: 0,
	}
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
func (rt *RuntimeTrigger) Type() Type {
	return RuntimeType
}

// Advance increments the AdvanceCount
func (rt *RuntimeTrigger) Advance() error {
	rt.AdvanceCount++
	return nil
}

// Trigger sends the workflowID over the trigger channel
func (rt *RuntimeTrigger) Trigger(t chan string, workflowID string) {
	if t == nil {
		log.Debugf("RuntimeTrigger: given trigger channel is nil")
	}
	t <- workflowID
}

type runtimeTrigger struct {
	Active       bool `json:"active"`
	Type         Type `json:"type"`
	AdvanceCount int  `json:"advanceCount"`
}

// MarshalJSON implements the json.Marshaller interface
func (rt *RuntimeTrigger) MarshalJSON() ([]byte, error) {
	if rt == nil {
		rt = &RuntimeTrigger{}
	}
	return json.Marshal(runtimeTrigger{
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
	rt.active = t.Active
	rt.AdvanceCount = t.AdvanceCount
	if rt == nil {
		rt = &RuntimeTrigger{}
	}
	return nil
}

// RuntimeListener listens for RuntimeTriggers to fire
type RuntimeListener struct {
	bus                event.Bus
	TriggerCh          chan string
	listening          bool
	activeTriggersLock sync.Mutex
	activeTriggers     map[profile.ID][]string
}

// NewRuntimeListener creates a RuntimeListener, and begin receiving on the
// trigger channel. Any triggers received before the RuntimeListener has been
// started using `runtimeListener.Start(ctx)` will be ignored
func NewRuntimeListener(ctx context.Context, bus event.Bus) *RuntimeListener {
	rl := &RuntimeListener{
		bus:            bus,
		TriggerCh:      make(chan string),
		activeTriggers: map[profile.ID][]string{},
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
func (l *RuntimeListener) ConstructTrigger(opt *Options) (Trigger, error) {
	if opt.Type != l.Type() {
		return nil, fmt.Errorf("%w, expected %q but got %q", ErrTypeMismatch, l.Type(), opt.Type)
	}
	a, ok := opt.Config["active"]
	if !ok {
		a = false
	}
	active, ok := a.(bool)
	if !ok {
		return nil, fmt.Errorf("expected \"active\" field to be a boolean value")
	}
	c, ok := opt.Config["advanceCount"]
	if !ok {
		c = 0
	}
	count, ok := c.(int)
	if !ok {
		return nil, fmt.Errorf("expected \"advanceCount\" field to be an int value")
	}

	return &RuntimeTrigger{
		AdvanceCount: int(count),
		active:       active,
	}, nil
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
		scopeID := s.ScopeID()
		if scopeID == "" {
			return ErrEmptyScopeID
		}
		triggers := s.ActiveTriggers(RuntimeType)
		wids, ok := l.activeTriggers[scopeID]
		if !ok {
			if len(triggers) == 0 {
				continue
			}
			l.activeTriggers[scopeID] = []string{}
			wids = l.activeTriggers[scopeID]
		}
		index := -1
		for i, wid := range wids {
			if wid == workflowID {
				index = i
				break
			}
		}
		if len(triggers) == 0 {
			l.activeTriggers[scopeID] = remove(l.activeTriggers[scopeID], index)
			continue
		}
		if index == -1 {
			l.activeTriggers[scopeID] = append(wids, workflowID)
		}
	}
	return nil
}

func remove(wids []string, i int) []string {
	if i < 0 || i > len(wids)-1 {
		return wids
	}
	wids[i] = wids[len(wids)-1]
	return wids[:len(wids)-1]
}

// Bus returns the underlying `event.Bus` that the RuntimeListener will emit
// an `event.ETTriggerWorkflow` event
func (l *RuntimeListener) Bus() event.Bus {
	return l.bus
}

// Type returns the Type `RuntimeType`
func (l *RuntimeListener) Type() Type {
	return RuntimeType
}

func (l *RuntimeListener) start(ctx context.Context) error {
	go func() {
		for {
			select {
			case id := <-l.TriggerCh:
				if !l.listening {
					log.Debugf("RuntimeListener: trigger ignored")
					continue
				}
				if err := l.shouldTrigger(ctx, id); err != nil {
					log.Debugf("RuntimeListener error: %s", err)
					continue
				}

				err := l.bus.Publish(ctx, event.ETWorkflowTrigger, id)
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

func (l *RuntimeListener) shouldTrigger(ctx context.Context, id string) error {
	l.activeTriggersLock.Lock()
	defer l.activeTriggersLock.Unlock()

	for _, widsForOwner := range l.activeTriggers {
		for _, wid := range widsForOwner {
			if wid == id {
				return nil
			}
		}
	}
	return fmt.Errorf("no active Runtime trigger associated with the given workflow ID %q", id)
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

// TriggerExists returns true if there is a record of a trigger for the given workflow id
func (l *RuntimeListener) TriggerExists(source Source) bool {
	l.activeTriggersLock.Lock()
	defer l.activeTriggersLock.Unlock()
	ids, ok := l.activeTriggers[source.ScopeID()]
	if !ok {
		return false
	}
	for _, id := range ids {
		if id == source.WorkflowID() {
			return true
		}
	}
	return false
}
