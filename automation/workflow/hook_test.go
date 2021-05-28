package workflow

import (
	"encoding/json"

	"github.com/qri-io/qri/event"
)

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
	return ETTestHook, th.payload
}

type testHook struct {
	Enabled      bool        `json:"enabled"`
	HookType     HookType    `json:"type"`
	AdvanceCount int         `json:"advancedCount"`
	Payload      interface{} `json:"payload"`
}

func (th *TestHook) MarshalJSON() ([]byte, error) {
	if th == nil {
		th = &TestHook{}
	}
	return json.Marshal(testHook{
		Enabled:      th.enabled,
		HookType:     th.hookType,
		AdvanceCount: th.AdvanceCount,
		Payload:      th.payload,
	})
}

func (th *TestHook) UnmarshalJSON(d []byte) error {
	h := &testHook{}
	err := json.Unmarshal(d, h)
	if err != nil {
		return err
	}
	if th == nil {
		th = &TestHook{}
	}
	th.enabled = h.Enabled
	th.hookType = h.HookType
	th.AdvanceCount = h.AdvanceCount
	th.payload = h.Payload
	return nil
}
