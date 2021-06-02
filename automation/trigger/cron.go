package trigger

import (
	"context"
	"encoding/json"
	"time"

	"github.com/qri-io/iso8601"
	"github.com/qri-io/qri/event"
)

const CronTriggerType = Type("cron")

type CronTrigger struct {
	enabled      bool
	start        time.Time
	perodicity   iso8601.RepeatingInterval
	nextRunStart *time.Time
}

var _ Trigger = (*CronTrigger)(nil)

func (ct *CronTrigger) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.ToMap())
}

func (ct *CronTrigger) UnmarshalJSON(p []byte) error {
	v := struct {
		Enabled      bool
		Start        time.Time
		Perodicity   iso8601.RepeatingInterval
		NextRunStart *time.Time
	}{}

	if err := json.Unmarshal(p, &v); err != nil {
		return err
	}

	ct.enabled = v.Enabled
	ct.start = v.Start
	ct.perodicity = v.Perodicity
	ct.nextRunStart = v.NextRunStart
	return nil
}

func (ct *CronTrigger) Enabled() bool { return ct.enabled }
func (ct *CronTrigger) SetEnabled(enabled bool) error {
	ct.enabled = enabled
	return nil
}
func (ct *CronTrigger) Type() Type { return CronTriggerType }

func (ct *CronTrigger) ToMap() map[string]interface{} {
	v := map[string]interface{}{
		"enabled":     ct.enabled,
		"start":       ct.start.Format(time.RFC3339),
		"periodicity": ct.perodicity.String(),
	}

	if ct.nextRunStart != nil {
		v["nextRunStart"] = ct.nextRunStart.Format(time.RFC3339)
	}

	return v
}

func (ct *CronTrigger) Advance() error {
	return nil
}

type CronListener struct {
	pub      event.Publisher
	active   map[string][]*CronTrigger
	interval time.Duration
}

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

func (c *CronListener) Listen(sources ...Source) {
	for _, src := range sources {
		trigs := src.ActiveTriggers(CronTriggerType)
		if len(trigs) > 0 {
			cts := make([]*CronTrigger, len(trigs))
			for i, t := range trigs {
				if ct, ok := t.(*CronTrigger); ok {
					cts[i] = ct
				}
			}
			c.active[src.WorkflowIDString()] = cts
		}
	}
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
	return nil
}

func (c *CronListener) Stop() error {
	return nil
}
