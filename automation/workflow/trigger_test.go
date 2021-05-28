package workflow

import "encoding/json"

// A TestTrigger implements the Trigger interface & keeps track of the number
// of times it had been advanced
type TestTrigger struct {
	enabled      bool        `json:"enabled"`
	triggerType  TriggerType `json:"type"`
	AdvanceCount int         `json:"advanceCount"`
}

var _ Trigger = (*TestTrigger)(nil)

// TestTriggerType denotes a `TestTrigger`
var TestTriggerType = TriggerType("Test Trigger")

// NewTestTrigger returns an enabled `TestTrigger`
func NewTestTrigger() *TestTrigger {
	return &TestTrigger{
		enabled:      true,
		triggerType:  TestTriggerType,
		AdvanceCount: 0,
	}
}

func (tt *TestTrigger) Enabled() bool {
	return tt.enabled
}

func (tt *TestTrigger) SetEnabled(enabled bool) error {
	tt.enabled = enabled
	return nil
}

func (tt *TestTrigger) Type() TriggerType {
	return tt.triggerType
}

func (tt *TestTrigger) Advance() error {
	tt.AdvanceCount++
	return nil
}

func (tt *TestTrigger) MarshalJSON() ([]byte, error) {
	return json.Marshal(tt)
}

func (tt *TestTrigger) UnmarshalJSON(d []byte) error {
	if tt == nil {
		tt = &TestTrigger{}
	}
	return json.Unmarshal(d, tt)
}
