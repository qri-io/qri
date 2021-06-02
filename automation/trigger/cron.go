package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/qri-io/iso8601"
	"github.com/qri-io/qri/event"
)

const CronTriggerType = "cron"

type CronTrigger struct {
	active       bool
	start        time.Time
	periodicity  iso8601.RepeatingInterval
	nextRunStart *time.Time
}

var _ Trigger = (*CronTrigger)(nil)

func NewCronTrigger(vals map[string]interface{}) (*CronTrigger, error) {
	data, err := json.Marshal(vals)
	if err != nil {
		return nil, err
	}
	ct := &CronTrigger{}
	err = ct.UnmarshalJSON(data)
	return ct, err
}

func (ct *CronTrigger) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.ToMap())
}

func (ct *CronTrigger) UnmarshalJSON(p []byte) error {
	v := struct {
		Active       bool
		Start        time.Time
		Perodicity   iso8601.RepeatingInterval
		NextRunStart *time.Time
	}{}

	if err := json.Unmarshal(p, &v); err != nil {
		return err
	}

	ct.active = v.Active
	ct.start = v.Start
	ct.periodicity = v.Perodicity
	ct.nextRunStart = v.NextRunStart
	return nil
}

func (ct *CronTrigger) Active() bool { return ct.active }
func (ct *CronTrigger) SetActive(active bool) error {
	ct.active = active
	return nil
}
func (CronTrigger) Type() string { return CronTriggerType }

func (ct *CronTrigger) ToMap() map[string]interface{} {
	v := map[string]interface{}{
		"active":      ct.active,
		"start":       ct.start.Format(time.RFC3339),
		"periodicity": ct.periodicity.String(),
	}

	if ct.nextRunStart != nil {
		v["nextRunStart"] = ct.nextRunStart.Format(time.RFC3339)
	}

	return v
}

func (ct *CronTrigger) Advance() error {
	ct.periodicity = ct.periodicity.NextRep()
	if ct.nextRunStart != nil {
		*ct.nextRunStart = ct.periodicity.After(*ct.nextRunStart)
	}
	*ct.nextRunStart = ct.periodicity.After(time.Now())
	return nil
}

type CronListener struct {
	pub      event.Publisher
	active   map[string][]*CronTrigger
	interval time.Duration
}

var _ Listener = (*CronListener)(nil)

const DefaultInterval = time.Second

func NewCronListener(pub event.Publisher) *CronListener {
	return NewCronListenerInterval(pub, DefaultInterval)
}

func NewCronListenerInterval(pub event.Publisher, interval time.Duration) *CronListener {
	return &CronListener{
		pub:      pub,
		active:   map[string][]*CronTrigger{},
		interval: interval,
	}
}

func (c *CronListener) Type() string { return CronTriggerType }

func (c *CronListener) ConstructTrigger(cfg map[string]interface{}) (Trigger, error) {
	typ := cfg["type"]
	if typ != c.Type() {
		return nil, fmt.Errorf("%w, expected %q but got %q", ErrTypeMismatch, c.Type(), typ)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	trig := &CronTrigger{}
	err = trig.UnmarshalJSON(data)
	return trig, err
}

func (c *CronListener) Listen(sources ...Source) error {
	for _, src := range sources {
		trigs := src.ActiveTriggers(CronTriggerType)
		if len(trigs) > 0 {
			cts := make([]*CronTrigger, len(trigs))
			for i, t := range trigs {
				if ct, ok := t.(*CronTrigger); ok {
					cts[i] = ct
				}
			}
			c.active[src.WorkflowID()] = cts
		}
	}

	return nil
}

func (c *CronListener) Start(ctx context.Context) error {
	check := func(ctx context.Context) {
		now := time.Now()
		for wid, ts := range c.active {
			for _, t := range ts {
				if t.nextRunStart != nil && now.After(*t.nextRunStart) {
					// run!
					c.pub.Publish(ctx, event.ETWorkflowTrigger, wid)
					if err := t.Advance(); err != nil {
						// TODO(b5): print error
					}
				}
			}
		}
	}

	t := time.NewTicker(c.interval)
	for {
		select {
		case <-t.C:
			check(ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *CronListener) Stop() error {
	return nil
}
