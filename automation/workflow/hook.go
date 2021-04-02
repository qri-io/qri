package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// HookType describes the type of hook
type HookType string

var (
	// HTPush indicates a "push" hook type, which will push the dataset to a remote after a successful run
	HTPush HookType = "push"
)

// Hook is the generic interface for a type of hook
type Hook interface {
	// ToMap converts a hook into a map[string]interface
	ToMap() map[string]interface{}
	// Advance takes a workflow and updates necessary fields on the Hook to set
	// the hook up correctly before it is run
	Advance(workflow *Workflow) error
}

// HookInfo contains the information needed to run a hook after a workflow
// successfully run
type HookInfo struct {
	// configuration
	ID         string   `json:"id"` // identifier
	WorkflowID string   `json:"workflowId"`
	Type       HookType `json:"type"`
	Disabled   bool     `json:"disabled"`

	// orchestration state
	RunCount  int    `json:"runCount"`
	LastRunID string `json:"lastRunID,omitempty"`
}

func newHookInfo(m map[string]interface{}) (HookInfo, error) {
	hi := HookInfo{}
	if id, ok := m["id"].(string); ok && id != "" {
		hi.ID = id
	} else {
		hi.ID = newHookID()
	}
	if wid, ok := m["workflowId"].(string); ok {
		hi.WorkflowID = wid
	}
	if t, ok := m["type"].(string); ok {
		hi.Type = HookType(t)
	} else {
		return HookInfo{}, fmt.Errorf("trigger type field is required")
	}
	if c, ok := m["runCount"].(int); ok {
		hi.RunCount = c
	}
	if d, ok := m["disabled"].(bool); ok {
		hi.Disabled = d
	}

	return hi, nil
}

// Info returns a HookInfo
func (hi HookInfo) Info() HookInfo {
	return hi
}

func (hi HookInfo) toMap() map[string]interface{} {
	return map[string]interface{}{
		"id":         hi.ID,
		"workflowId": hi.WorkflowID,
		"type":       hi.Type,
		"disabled":   hi.Disabled,
		"runCount":   hi.RunCount,
		"lastRunID":  hi.LastRunID,
	}
}

// Hooks is a slice of Hook
type Hooks []Hook

var (
	_ json.Marshaler   = (*Hooks)(nil)
	_ json.Unmarshaler = (*Hooks)(nil)
)

// MarshalJSON serializes `Hooks` to a JSON array of `Hook`s
func (hs Hooks) MarshalJSON() ([]byte, error) {
	items := make([]map[string]interface{}, len(hs))
	for i, h := range hs {
		items[i] = h.ToMap()
	}
	return json.Marshal(items)
}

// UnmarshalJSON deserializes from a JSON array
func (hs *Hooks) UnmarshalJSON(data []byte) error {
	mapped := []map[string]interface{}{}
	if err := json.Unmarshal(data, &mapped); err != nil {
		return err
	}

	hooks := make(Hooks, 0, len(mapped))
	for _, m := range mapped {
		t, err := newHookFromMap(m)
		if err != nil {
			return err
		}
		hooks = append(hooks, t)
	}
	*hs = hooks
	return nil
}

func newHookFromMap(m map[string]interface{}) (Hook, error) {
	hi, err := newHookInfo(m)
	if err != nil {
		return nil, err
	}

	switch hi.Type {
	case HTPush:
		h := &PushHook{HookInfo: hi}
		reg, ok := m["remote"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid push hook. expected %s to be a string type", "remote")
		}
		h.Remote = reg
		return h, nil
	default:
		return nil, fmt.Errorf("unknown hook type %q", hi.Type)
	}
}

func newHookID() string {
	return uuid.New().String()
}

// PushHook represents a specific Hook meant to push a dataset to a remote
type PushHook struct {
	HookInfo
	Remote string `json:"remote"`
}

// NewPushHook creates a PushHook
func NewPushHook(workflowID string, remote string) *PushHook {
	return &PushHook{
		HookInfo: HookInfo{
			ID:         newHookID(),
			WorkflowID: workflowID,
			Type:       HTPush,
		},
		Remote: remote,
	}
}

// ToMap maps a PushHook to a map[string]interface
func (ph *PushHook) ToMap() map[string]interface{} {
	m := ph.toMap()
	m["remote"] = ph.Remote
	return m
}

// Advance increments the run count and records the run id that triggered
// the hook to run
func (ph *PushHook) Advance(workflow *Workflow) error {
	ph.RunCount++
	ph.LastRunID = workflow.CurrentRun.ID
	return nil
}
