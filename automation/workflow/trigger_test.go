package workflow

import (
	"encoding/json"
)

// A TestTrigger implements the Trigger interface & keeps track of the number
// of times it had been advanced
type TestTrigger struct {
	enabled      bool
	triggerType  TriggerType
	AdvanceCount int
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

type testTrigger struct {
	Enabled      bool        `json:"enabled"`
	TriggerType  TriggerType `json:"type"`
	AdvanceCount int         `json:"advanceCount"`
}

func (tt *TestTrigger) MarshalJSON() ([]byte, error) {
	if tt == nil {
		tt = &TestTrigger{}
	}
	return json.Marshal(testTrigger{
		Enabled:      tt.enabled,
		TriggerType:  tt.triggerType,
		AdvanceCount: tt.AdvanceCount,
	})
}

func (tt *TestTrigger) UnmarshalJSON(d []byte) error {
	t := &testTrigger{}
	err := json.Unmarshal(d, t)
	if err != nil {
		return err
	}
	tt.enabled = t.Enabled
	tt.triggerType = t.TriggerType
	tt.AdvanceCount = t.AdvanceCount
	if tt == nil {
		tt = &TestTrigger{}
	}
	return nil
}
