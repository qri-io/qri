package trigger

import (
	"encoding/json"
	"fmt"
)

// A RuntimeTrigger implements the Trigger interface & keeps track of the number
// of times it had been advanced
type RuntimeTrigger struct {
	enabled      bool
	AdvanceCount int
}

var _ Trigger = (*RuntimeTrigger)(nil)

// RuntimeType denotes a `RuntimeTrigger`
var RuntimeType = Type("Runtime Trigger")

// NewRuntimeTrigger returns an enabled `RuntimeTrigger`
func NewRuntimeTrigger() *RuntimeTrigger {
	return &RuntimeTrigger{
		enabled:      true,
		AdvanceCount: 0,
	}
}

// Enabled returns if the RuntimeTrigger is enabled
func (rt *RuntimeTrigger) Enabled() bool {
	return rt.enabled
}

// SetEnabled sets the enabled status
func (rt *RuntimeTrigger) SetEnabled(enabled bool) error {
	rt.enabled = enabled
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

type runtimeTrigger struct {
	Enabled      bool `json:"enabled"`
	Type         Type `json:"type"`
	AdvanceCount int  `json:"advanceCount"`
}

// MarshalJSON implements the json.Marshaller interface
func (rt *RuntimeTrigger) MarshalJSON() ([]byte, error) {
	if rt == nil {
		rt = &RuntimeTrigger{}
	}
	return json.Marshal(runtimeTrigger{
		Enabled:      rt.enabled,
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
	rt.enabled = t.Enabled
	rt.AdvanceCount = t.AdvanceCount
	if rt == nil {
		rt = &RuntimeTrigger{}
	}
	return nil
}

func (rt *RuntimeTrigger) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":         string(RuntimeType),
		"enabled":      rt.enabled,
		"advanceCount": rt.AdvanceCount,
	}
}
