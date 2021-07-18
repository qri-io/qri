package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/qri-io/iso8601"
	"github.com/qri-io/qri/event"
)

// NowFunc returns the time now
// can be overridded for testing purposes
var NowFunc = func() time.Time {
	return time.Now()
}

// CronTrigger implements the Trigger interface & keeps track of periodicity
// and the next run time
type CronTrigger struct {
	id           string
	active       bool
	periodicity  iso8601.RepeatingInterval
	nextRunStart *time.Time
}

var _ Trigger = (*CronTrigger)(nil)

// CronType denotes a `CronTrigger`
const CronType = "cron"

// NewCronTrigger returns an active `CronTrigger`
func NewCronTrigger(vals map[string]interface{}) (*CronTrigger, error) {
	data, err := json.Marshal(vals)
	if err != nil {
		return nil, err
	}
	ct := &CronTrigger{}
	err = ct.UnmarshalJSON(data)
	if ct.nextRunStart == nil {
		ct.nextRunStart = ct.periodicity.Interval.Start
	}
	ct.active = true
	return ct, err
}

// ID returns the trigger.ID
func (ct *CronTrigger) ID() string { return ct.id }

// Active returns true if the CronTrigger is active
func (ct *CronTrigger) Active() bool { return ct.active }

// SetActive sets the active status
func (ct *CronTrigger) SetActive(active bool) error {
	ct.active = active
	return nil
}

// Type returns the CronType
func (CronTrigger) Type() string { return CronType }

// Advance sets the periodicity and nextRunStart to be ready for the next run
func (ct *CronTrigger) Advance() error {
	ct.periodicity = ct.periodicity.NextRep()
	if ct.nextRunStart != nil {
		*ct.nextRunStart = ct.periodicity.After(*ct.nextRunStart)
		return nil
	}
	*ct.nextRunStart = ct.periodicity.After(NowFunc())
	return nil
}

// ToMap returns the trigger as a map[string]interface{}
func (ct *CronTrigger) ToMap() map[string]interface{} {
	v := map[string]interface{}{
		"id":          ct.id,
		"active":      ct.active,
		"periodicity": ct.periodicity.String(),
		"type":        CronType,
	}

	if ct.nextRunStart != nil {
		v["nextRunStart"] = ct.nextRunStart.Format(time.RFC3339)
	}

	return v
}

// MarshalJSON satisfies the json.Marshaller interface
func (ct *CronTrigger) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.ToMap())
}

// UnmarshalJSON satisfies the json.Unmarshaller interface
func (ct *CronTrigger) UnmarshalJSON(p []byte) error {
	v := struct {
		Type         string     `json:"type"`
		ID           string     `json:"id"`
		Active       bool       `json:"active"`
		Start        time.Time  `json:"start"`
		Periodicity  string     `json:"periodicity"`
		NextRunStart *time.Time `json:"nextRunStart"`
	}{}

	if err := json.Unmarshal(p, &v); err != nil {
		return err
	}
	if v.Type != CronType {
		return ErrUnexpectedType
	}

	ct.id = v.ID
	ct.active = v.Active
	periodicity, err := iso8601.ParseRepeatingInterval(v.Periodicity)
	if err != nil {
		return err
	}
	ct.periodicity = periodicity
	ct.nextRunStart = v.NextRunStart
	return nil
}

// CronListener listens for CronTriggers
type CronListener struct {
	cancel       context.CancelFunc
	pub          event.Publisher
	interval     time.Duration
	triggerStore *MemTriggerStore
}

var _ Listener = (*CronListener)(nil)

// DefaultInterval is the default amount of time to wait before checking
// if any CronTriggers have fired
const DefaultInterval = time.Second

// NewCronListener returns a CronListener with the DefaultInterval
func NewCronListener(pub event.Publisher) *CronListener {
	return NewCronListenerInterval(pub, DefaultInterval)
}

// NewCronListenerInterval returns a CronListener with the given interval
func NewCronListenerInterval(pub event.Publisher, interval time.Duration) *CronListener {
	cl := &CronListener{
		pub:      pub,
		interval: interval,
	}
	cl.triggerStore = NewMemTriggerStore(cl)
	return cl
}

// ConstructTrigger returns a CronTrigger from a map[string]interface{}
func (c *CronListener) ConstructTrigger(cfg map[string]interface{}) (Trigger, error) {
	typ := cfg["type"]
	if typ != c.Type() {
		return nil, fmt.Errorf("%w, expected %q but got %q", ErrTypeMismatch, c.Type(), typ)
	}

	_, ok := cfg["periodicity"]
	if !ok {
		return nil, fmt.Errorf("field %q required", "periodicity")
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	trig := &CronTrigger{}
	err = trig.UnmarshalJSON(data)
	if trig.id == "" {
		trig.id = NewID()
	}
	if trig.nextRunStart == nil {
		trig.nextRunStart = trig.periodicity.Interval.Start
	}
	return trig, err
}

// Listen takes a list of sources and adds or updates the Listener's store to
// include all the active triggers of the CronType
func (c *CronListener) Listen(sources ...Source) error {
	return c.triggerStore.Put(sources...)
}

// Type returns the CronType
func (c *CronListener) Type() string { return CronType }

// Start tells the CronListener to begin listening for CronTriggers
func (c *CronListener) Start(ctx context.Context) error {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	check := func(ctx context.Context) {
		now := NowFunc()
		for ownerID, wids := range c.triggerStore.Active() {
			for workflowID, triggers := range wids {
				for _, trig := range triggers {
					t := trig.(*CronTrigger)
					if t.nextRunStart != nil && now.After(*t.nextRunStart) {
						wte := event.WorkflowTriggerEvent{
							WorkflowID: workflowID,
							OwnerID:    ownerID,
							TriggerID:  t.ID(),
						}
						if err := c.pub.Publish(ctx, event.ETAutomationWorkflowTrigger, wte); err != nil {
							log.Debugw("CronListener: publish ETAutomationWorkflowTrigger", "error", err, "WorkflowTriggerEvent", wte)
						}
					}
				}
			}
		}
	}

	go func() {
		t := time.NewTicker(c.interval)
		for {
			select {
			case <-t.C:
				check(ctx)
			case <-ctxWithCancel.Done():
				return
			}
		}
	}()
	return nil
}

// Stop tells the CronListener to stop listening for CronTriggers
func (c *CronListener) Stop() error {
	c.cancel()
	return nil
}
