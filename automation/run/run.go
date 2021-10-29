// Package run defines metadata about transform script execution
package run

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/google/uuid"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/event"
)

var (
	log = golog.Logger("run")
	// ErrNoID indicates the run.State has no run ID
	ErrNoID = fmt.Errorf("no run ID")
	// ErrNoWorkflowID indicates the run.State has no workflow.ID
	ErrNoWorkflowID = fmt.Errorf("no workflow ID")
)

// NewID creates a run identifier
func NewID() string {
	return uuid.New().String()
}

// SetIDRand sets the random reader that NewID uses as a source of random bytes
// passing in nil will default to crypto.Rand. This can be used to make ID
// generation deterministic for tests. eg:
//    myString := "SomeRandomStringThatIsLong-SoYouCanCallItAsMuchAsNeeded..."
//    run.SetIDRand(strings.NewReader(myString))
//    a := NewID()
//    run.SetIDRand(strings.NewReader(myString))
//    b := NewID()
func SetIDRand(r io.Reader) {
	uuid.SetRand(r)
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
	ID         string       `json:"id"`
	WorkflowID workflow.ID  `json:"workflowID"`
	Number     int          `json:"number"`
	Status     Status       `json:"status"`
	Message    string       `json:"message"`
	StartTime  *time.Time   `json:"startTime"`
	StopTime   *time.Time   `json:"stopTime"`
	Duration   int64        `json:"duration"`
	Steps      []*StepState `json:"steps"`
}

// NewState returns a new *State with the given runID
func NewState(runID string) *State {
	return &State{ID: runID}
}

// Validate errors if the run is not valid
func (rs *State) Validate() error {
	if rs.ID == "" {
		return ErrNoID
	}
	if rs.WorkflowID.String() == "" {
		return ErrNoWorkflowID
	}
	return nil
}

// Copy returns a shallow copy of the receiver
func (rs *State) Copy() *State {
	if rs == nil {
		return nil
	}
	run := &State{
		ID:         rs.ID,
		WorkflowID: rs.WorkflowID,
		Number:     rs.Number,
		Status:     rs.Status,
		Message:    rs.Message,
		StartTime:  rs.StartTime,
		StopTime:   rs.StopTime,
		Duration:   rs.Duration,
		Steps:      rs.Steps,
	}
	return run
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
			rs.Duration = int64(rs.StopTime.Sub(*rs.StartTime))
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
			step.Duration = int64(step.StopTime.Sub(*step.StartTime))
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
	case event.ETTransformCanceled:
		return nil
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
	Duration  int64         `json:"duration"`
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

type _stepState struct {
	Name      string     `json:"name"`
	Category  string     `json:"category"`
	Status    Status     `json:"status"`
	StartTime *time.Time `json:"startTime"`
	StopTime  *time.Time `json:"stopTime"`
	Duration  int64      `json:"duration"`
	Output    []_event   `json:"output"`
}

type _event struct {
	Type      event.Type      `json:"type"`
	Timestamp int64           `json:"timestamp"`
	SessionID string          `json:"sessionID"`
	Payload   json.RawMessage `json:"payload"`
}

// UnmarshalJSON satisfies the json.Unmarshaller interface
func (ss *StepState) UnmarshalJSON(data []byte) error {
	tmpSS := &StepState{}
	rs := &_stepState{}
	if err := json.Unmarshal(data, rs); err != nil {
		return err
	}
	tmpSS.Name = rs.Name
	tmpSS.Category = rs.Category
	tmpSS.Status = rs.Status
	tmpSS.StartTime = rs.StartTime
	tmpSS.StopTime = rs.StopTime
	tmpSS.Duration = rs.Duration
	for _, re := range rs.Output {
		e := event.Event{
			Type:      re.Type,
			Timestamp: re.Timestamp,
			SessionID: re.SessionID,
		}
		switch e.Type {
		case event.ETTransformStart, event.ETTransformStop:
			p := event.TransformLifecycle{}
			if err := json.Unmarshal(re.Payload, &p); err != nil {
				return err
			}
			e.Payload = p
		case event.ETTransformStepStart, event.ETTransformStepStop, event.ETTransformStepSkip:
			p := event.TransformStepLifecycle{}
			if err := json.Unmarshal(re.Payload, &p); err != nil {
				return err
			}
			e.Payload = p
		case event.ETTransformPrint, event.ETTransformError:
			p := event.TransformMessage{}
			if err := json.Unmarshal(re.Payload, &p); err != nil {
				return err
			}
			e.Payload = p
		case event.ETTransformDatasetPreview:
			e.Payload = &dataset.Dataset{}
			if err := json.Unmarshal(re.Payload, e.Payload); err != nil {
				return err
			}
		default:
			if err := json.Unmarshal(re.Payload, e.Payload); err != nil {
				return err
			}
		}
		tmpSS.Output = append(tmpSS.Output, e)
	}
	*ss = *tmpSS
	return nil
}

func toTimePointer(unixnano int64) *time.Time {
	t := time.Unix(0, unixnano)
	return &t
}

// Set is a collection of run.States that implements the sort.Interface,
// sorting a list of run.State in reverse-chronological order
type Set struct {
	set []*State
}

// NewSet constructs a run.State set
func NewSet() *Set {
	return &Set{}
}

// Len is part of the sort.Interface
func (s Set) Len() int { return len(s.set) }

// Less is part of the `sort.Interface`
func (s Set) Less(i, j int) bool {
	return lessNilTime(s.set[i].StartTime, s.set[j].StartTime)
}

// Swap is part of the `sort.Interface`
func (s Set) Swap(i, j int) { s.set[i], s.set[j] = s.set[j], s.set[i] }

// Add adds a run.State to a Set
func (s *Set) Add(j *State) {
	if s == nil {
		*s = Set{set: []*State{j}}
		return
	}

	for i, run := range s.set {
		if run.ID == j.ID {
			s.set[i] = j
			return
		}
	}
	s.set = append(s.set, j)
	sort.Sort(s)
}

// Remove removes a run.State from a Set
func (s *Set) Remove(id string) (removed bool) {
	for i, run := range s.set {
		if run.ID == id {
			if i+1 == len(s.set) {
				s.set = s.set[:i]
				return true
			}

			s.set = append(s.set[:i], s.set[i+1:]...)
			return true
		}
	}
	return false
}

// Slice returns a slice of run.States from position `start` to position `end`
func (s *Set) Slice(start, end int) []*State {
	if start < 0 || end < 0 {
		return []*State{}
	}
	if end > s.Len() {
		end = s.Len()
	}
	return s.set[start:end]
}

// MarshalJSON satisfies the `json.Marshaller` interface
func (s Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.set)
}

// UnmarshalJSON satisfies the `json.Unmarshaller` interface
func (s *Set) UnmarshalJSON(data []byte) error {
	set := []*State{}
	if err := json.Unmarshal(data, &set); err != nil {
		return err
	}
	s.set = set
	return nil
}

func lessNilTime(a, b *time.Time) bool {
	if a == nil && b != nil {
		return true
	} else if a != nil && b == nil {
		return false
	} else if a == nil && b == nil {
		return false
	}
	return a.After(*b)
}
