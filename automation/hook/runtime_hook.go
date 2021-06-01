package hook

import (
	"encoding/json"

	"github.com/qri-io/qri/event"
)

// A RuntimeHook implements rhe Hook interface & keeps track of rhe number
// of times it had been advanced
type RuntimeHook struct {
	enabled      bool
	hookType     HookType
	AdvanceCount int
	event        event.Type
	payload      interface{}
}

var _ Hook = (*RuntimeHook)(nil)

// RuntimeHookType denotes a `RuntimeHook`
var RuntimeHookType = HookType("Runtime Hook")

// ETRuntimeHook denotes a `RuntimeHook` event
// TODO (ramfox): this will probably move to the `event` package
const ETRuntimeHook = event.Type("workflow:runtimehook")

// NewRuntimeHook returns an enabled `RuntimeHook`
func NewRuntimeHook(payload interface{}) *RuntimeHook {
	return &RuntimeHook{
		enabled:      true,
		hookType:     RuntimeHookType,
		AdvanceCount: 0,
		payload:      payload,
	}
}

func (rh *RuntimeHook) Enabled() bool {
	return rh.enabled
}

func (rh *RuntimeHook) SetEnabled(enabled bool) error {
	rh.enabled = enabled
	return nil
}

func (rh *RuntimeHook) Type() HookType {
	return rh.hookType
}

func (rh *RuntimeHook) Advance() error {
	rh.AdvanceCount++
	return nil
}

func (rh *RuntimeHook) Event() (event.Type, interface{}) {
	return ETRuntimeHook, rh.payload
}

type runtimeHook struct {
	Enabled      bool        `json:"enabled"`
	HookType     HookType    `json:"type"`
	AdvanceCount int         `json:"advancedCount"`
	Payload      interface{} `json:"payload"`
}

func (rh *RuntimeHook) MarshalJSON() ([]byte, error) {
	if rh == nil {
		rh = &RuntimeHook{}
	}
	return json.Marshal(runtimeHook{
		Enabled:      rh.enabled,
		HookType:     rh.hookType,
		AdvanceCount: rh.AdvanceCount,
		Payload:      rh.payload,
	})
}

func (rh *RuntimeHook) UnmarshalJSON(d []byte) error {
	h := &runtimeHook{}
	err := json.Unmarshal(d, h)
	if err != nil {
		return err
	}
	if rh == nil {
		rh = &RuntimeHook{}
	}
	rh.enabled = h.Enabled
	rh.hookType = h.HookType
	rh.AdvanceCount = h.AdvanceCount
	rh.payload = h.Payload
	return nil
}
