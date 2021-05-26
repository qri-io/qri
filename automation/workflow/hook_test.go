package workflow

import "github.com/qri-io/qri/event"

// A TestHook implements the Hook interface & keeps track of the number
// of times it had been advanced
type TestHook struct {
	enabled      bool
	hookType     HookType
	AdvanceCount int
	event        event.Type
	payload      interface{}
}

var _ Hook = (*TestHook)(nil)

// TestHookType denotes a `TestHook`
var TestHookType = HookType("Test Hook")

// ETTestHook denotes a `TestHook` event
const ETTestHook = event.Type("workflow test:hook")

// NewTestHook returns an enabled `TestHook`
func NewTestHook(payload interface{}) *TestHook {
	return &TestHook{
		enabled:      true,
		hookType:     TestHookType,
		event:        ETTestHook,
		AdvanceCount: 0,
		payload:      payload,
	}
}

func (th *TestHook) Enabled() bool {
	return th.enabled
}

func (th *TestHook) SetEnabled(enabled bool) error {
	th.enabled = enabled
	return nil
}

func (th *TestHook) Type() HookType {
	return th.hookType
}

func (th *TestHook) Advance() error {
	th.AdvanceCount++
	return nil
}

func (th *TestHook) Event() (event.Type, interface{}) {
	return th.event, th.payload
}
