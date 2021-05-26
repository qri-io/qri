package workflow

// A TestTrigger implements the Trigger interface & keeps track of the number
// of times it had been advanced
type TestTrigger struct {
	workflowID   ID
	enabled      bool
	triggerType  TriggerType
	AdvanceCount int
}

var _ Trigger = (*TestTrigger)(nil)

// TestTriggerType denotes a `TestTrigger`
var TestTriggerType = TriggerType("Test Trigger")

// NewTestTrigger returns an enabled `TestTrigger`
func NewTestTrigger(wid ID) *TestTrigger {
	return &TestTrigger{
		workflowID:   wid,
		enabled:      true,
		triggerType:  TestTriggerType,
		AdvanceCount: 0,
	}
}

func (tt *TestTrigger) Enabled() bool {
	return tt.enabled
}

func (tt *TestTrigger) SetEnable(enabled bool) error {
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

func (tt *TestTrigger) WorkflowID() ID {
	return tt.workflowID
}
