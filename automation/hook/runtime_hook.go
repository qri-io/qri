package hook

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/qri/event"
)

// A RuntimeHook implements rhe Hook interface & keeps track of rhe number
// of times it had been advanced
type RuntimeHook struct {
	enabled      bool
	AdvanceCount int
	payload      interface{}
}

var _ Hook = (*RuntimeHook)(nil)

// RuntimeType denotes a `RuntimeHook`
var RuntimeType = Type("Runtime Hook")

// ETRuntimeHook denotes a `RuntimeHook` event
// TODO (ramfox): this will probably move to the `event` package
const ETRuntimeHook = event.Type("workflow:runtimehook")

// NewRuntimeHook returns an enabled `RuntimeHook`
func NewRuntimeHook(payload interface{}) *RuntimeHook {
	return &RuntimeHook{
		enabled:      true,
		AdvanceCount: 0,
		payload:      payload,
	}
}

// Enabled returns the enabled status
func (rh *RuntimeHook) Enabled() bool {
	return rh.enabled
}

// SetEnabled sets the enabled status
func (rh *RuntimeHook) SetEnabled(enabled bool) error {
	rh.enabled = enabled
	return nil
}

// Type returns the Type
func (rh *RuntimeHook) Type() Type {
	return RuntimeType
}

// Advance increments the AdvanceCount
func (rh *RuntimeHook) Advance() error {
	rh.AdvanceCount++
	return nil
}

// Event returns the event.Type ETRuntimeHook as well as the associated payload
func (rh *RuntimeHook) Event() (event.Type, interface{}) {
	return ETRuntimeHook, rh.payload
}

type runtimeHook struct {
	Enabled      bool        `json:"enabled"`
	Type         Type        `json:"type"`
	AdvanceCount int         `json:"advancedCount"`
	Payload      interface{} `json:"payload"`
}

// MarshalJSON satisfies the json.Marshaller interface
func (rh *RuntimeHook) MarshalJSON() ([]byte, error) {
	if rh == nil {
		rh = &RuntimeHook{}
	}
	return json.Marshal(runtimeHook{
		Enabled:      rh.enabled,
		Type:         rh.Type(),
		AdvanceCount: rh.AdvanceCount,
		Payload:      rh.payload,
	})
}

// UnmarshalJSON satisfies the json.Unmarshaller interface
func (rh *RuntimeHook) UnmarshalJSON(d []byte) error {
	h := &runtimeHook{}
	err := json.Unmarshal(d, h)
	if err != nil {
		return err
	}
	if h.Type != RuntimeType {
		return fmt.Errorf("%w, got %q expected %q", ErrUnexpectedType, h.Type, RuntimeType)
	}
	if rh == nil {
		rh = &RuntimeHook{}
	}
	rh.enabled = h.Enabled
	rh.AdvanceCount = h.AdvanceCount
	rh.payload = h.Payload
	return nil
}
