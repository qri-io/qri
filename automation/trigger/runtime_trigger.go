package trigger

import (
	"encoding/json"
)

// A RuntimeTrigger implements the Trigger interface & keeps track of the number
// of times it had been advanced
type RuntimeTrigger struct {
	enabled      bool
	triggerType  TriggerType
	AdvanceCount int
}

var _ Trigger = (*RuntimeTrigger)(nil)

// RuntimeTriggerType denotes a `RuntimeTrigger`
var RuntimeTriggerType = TriggerType(" Trigger")

// NewRuntimeTrigger returns an enabled `RuntimeTrigger`
func NewRuntimeTrigger() *RuntimeTrigger {
	return &RuntimeTrigger{
		enabled:      true,
		triggerType:  RuntimeTriggerType,
		AdvanceCount: 0,
	}
}

func (rt *RuntimeTrigger) Enabled() bool {
	return rt.enabled
}

func (rt *RuntimeTrigger) SetEnabled(enabled bool) error {
	rt.enabled = enabled
	return nil
}

func (rt *RuntimeTrigger) Type() TriggerType {
	return rt.triggerType
}

func (rt *RuntimeTrigger) Advance() error {
	rt.AdvanceCount++
	return nil
}

type runtimeTrigger struct {
	Enabled      bool        `json:"enabled"`
	TriggerType  TriggerType `json:"type"`
	AdvanceCount int         `json:"advanceCount"`
}

func (rt *RuntimeTrigger) MarshalJSON() ([]byte, error) {
	if rt == nil {
		rt = &RuntimeTrigger{}
	}
	return json.Marshal(runtimeTrigger{
		Enabled:      rt.enabled,
		TriggerType:  rt.triggerType,
		AdvanceCount: rt.AdvanceCount,
	})
}

func (rt *RuntimeTrigger) UnmarshalJSON(d []byte) error {
	t := &runtimeTrigger{}
	err := json.Unmarshal(d, t)
	if err != nil {
		return err
	}
	rt.enabled = t.Enabled
	rt.triggerType = t.TriggerType
	rt.AdvanceCount = t.AdvanceCount
	if rt == nil {
		rt = &RuntimeTrigger{}
	}
	return nil
}
