// Package run defines metadata about transform script execution
package run

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/qri-io/qri/event"
)

// NewID creates a run identifier
func NewID() string {
	return uuid.New().String()
}

// Status enumerates all possible execution states of a transform script or
// step within a script, in relation to the current time.
// Scripts & steps that have completed are broken into categories based on exit
// state
type Status string

const (
	// RSWaiting indicates a script/step that has yet to start
	RSWaiting = Status("waiting")
	// RSRunning indicates a script/step is currently executing
	RSRunning = Status("running")
	// RSSucceeded indicates a script/step has completed without error
	RSSucceeded = Status("succeeded")
	// RSFailed indicates a script/step completed & exited when an unexpected error
	// occured
	RSFailed = Status("failed")
	// RSUnchanged indicates a script completed but no changes were found
	// since the last version of the script succeeded
	RSUnchanged = Status("unchanged")
	// RSSkipped indicates a script/step was not executed
	RSSkipped = Status("skipped")
)

// State is a passable, cachable data structure that describes the execution of
// a transform. State structs can act as a sink of transform events, collapsing
// the state transition of multiple transform events into a single structure
type State struct {
	ID        string       `json:"id"`
	Number    int          `json:"number"`
	Status    Status       `json:"status"`
	Message   string       `json:"message"`
	StartTime *time.Time   `json:"startTime"`
	StopTime  *time.Time   `json:"stopTime"`
	Duration  int          `json:"duration"`
	Steps     []*StepState `json:"steps"`
}

// NewState is a simple constructor to remind package consumers that state
// structs must be initialized with an identifier to act as a sink of transform
// events
func NewState(id string) *State {
	return &State{
		ID: id,
	}
}

// AddTransformEvent alters state based on a given event
func (rs *State) AddTransformEvent(e event.Event) error {
	if rs.ID != e.SessionID {
		// silently ignore session ID mismatch
		return nil
	}

	switch e.Type {
	case event.ETTransformStart:
		rs.Status = RSRunning
		rs.StartTime = toTimePointer(e.Timestamp)
		return nil
	case event.ETTransformStop:
		rs.StopTime = toTimePointer(e.Timestamp)
		if tl, ok := e.Payload.(event.TransformLifecycle); ok {
			rs.Status = Status(tl.Status)
		}
		if rs.StartTime != nil && rs.StopTime != nil {
			rs.Duration = int(rs.StopTime.Sub(*rs.StartTime))
		}
		return nil
	case event.ETTransformStepStart:
		s, err := NewStepStateFromEvent(e)
		if err != nil {
			return err
		}
		s.Status = RSRunning
		s.StartTime = toTimePointer(e.Timestamp)
		rs.Steps = append(rs.Steps, s)
		return nil
	case event.ETTransformStepStop:
		step, err := rs.lastStep()
		if err != nil {
			return err
		}
		step.StopTime = toTimePointer(e.Timestamp)
		if tsl, ok := e.Payload.(event.TransformStepLifecycle); ok {
			step.Status = Status(tsl.Status)
		} else {
			step.Status = RSFailed
		}
		if step.StartTime != nil && step.StopTime != nil {
			step.Duration = int(step.StopTime.Sub(*step.StartTime))
		}
		return nil
	case event.ETTransformStepSkip:
		s, err := NewStepStateFromEvent(e)
		if err != nil {
			return err
		}
		s.Status = RSSkipped
		rs.Steps = append(rs.Steps, s)
		return nil
	case event.ETTransformPrint,
		event.ETTransformError,
		event.ETTransformDatasetPreview:
		return rs.appendStepOutputLog(e)
	}
	return fmt.Errorf("unexpected event type: %q", e.Type)
}

func (rs *State) lastStep() (*StepState, error) {
	if len(rs.Steps) > 0 {
		return rs.Steps[len(rs.Steps)-1], nil
	}
	return nil, fmt.Errorf("expected step to exist")
}

func (rs *State) appendStepOutputLog(e event.Event) error {
	step, err := rs.lastStep()
	if err != nil {
		return err
	}

	step.Output = append(step.Output, e)
	return nil
}

// StepState describes the execution of a transform step
type StepState struct {
	Name      string        `json:"name"`
	Category  string        `json:"category"`
	Status    Status        `json:"status"`
	StartTime *time.Time    `json:"startTime"`
	StopTime  *time.Time    `json:"stopTime"`
	Duration  int           `json:"duration"`
	Output    []event.Event `json:"output"`
}

// NewStepStateFromEvent constructs StepState from an event
func NewStepStateFromEvent(e event.Event) (*StepState, error) {
	if tsl, ok := e.Payload.(event.TransformStepLifecycle); ok {
		return &StepState{
			Name:     tsl.Name,
			Category: tsl.Category,
			Status:   Status(tsl.Status),
		}, nil
	}
	return nil, fmt.Errorf("run step event data must be a transform step lifecycle struct")
}

func toTimePointer(unixnano int64) *time.Time {
	t := time.Unix(0, unixnano)
	return &t
}
