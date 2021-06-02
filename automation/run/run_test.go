package run

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/event"
)

func ExampleNewID() {
	myString := "SomeRandomStringThatIsLong-SoYouCanCallItAsMuchAsNeeded..."
	SetIDRand(strings.NewReader(myString))
	a := NewID()
	SetIDRand(strings.NewReader(myString))
	b := NewID()

	fmt.Printf("a:  %s\nb:  %s\neq: %t", a, b, a == b)
	// Output:
	// a:  536f6d65-5261-4e64-af6d-537472696e67
	// b:  536f6d65-5261-4e64-af6d-537472696e67
	// eq: true
}

func TestStateAddTransformEvent(t *testing.T) {
	runID := NewID()
	states := []struct {
		e event.Event
		r *State
	}{
		{
			event.Event{Type: event.ETTransformStart, Timestamp: 1609460600090, SessionID: runID, Payload: event.TransformLifecycle{StepCount: 4, Status: "running"}},
			&State{ID: runID, StartTime: toTimePointer(1609460600090), Status: RSRunning},
		},
		{
			event.Event{Type: event.ETTransformStepStart, Timestamp: 1609460700090, SessionID: runID, Payload: event.TransformStepLifecycle{Name: "setup"}},
			&State{ID: runID, StartTime: toTimePointer(1609460600090), Status: RSRunning, Steps: []*StepState{
				{Name: "setup", StartTime: toTimePointer(1609460700090), Status: RSRunning},
			}},
		},
		// {
		//	event.Event{ Type: event.ETVersionPulled,           Timestamp: 1609460800090, SessionID: runID, Payload: {"refstring": "rico/presidents@QmFoo", "remote": "https://registy.qri.cloud" }},
		//	&State{},
		//	},
		{
			event.Event{Type: event.ETTransformStepStop, Timestamp: 1609460900090, SessionID: runID, Payload: event.TransformStepLifecycle{Name: "setup", Status: "succeeded"}},
			&State{ID: runID, StartTime: toTimePointer(1609460600090), Status: RSRunning, Steps: []*StepState{
				{Name: "setup", StartTime: toTimePointer(1609460700090), StopTime: toTimePointer(1609460900090), Duration: 200000, Status: RSSucceeded},
			}},
		},
		{
			event.Event{Type: event.ETTransformStepStart, Timestamp: 1609461000090, SessionID: runID, Payload: event.TransformStepLifecycle{Name: "download"}},
			&State{ID: runID, StartTime: toTimePointer(1609460600090), Status: RSRunning, Steps: []*StepState{
				{Name: "setup", StartTime: toTimePointer(1609460700090), StopTime: toTimePointer(1609460900090), Duration: 200000, Status: RSSucceeded},
				{Name: "download", StartTime: toTimePointer(1609461000090), Status: RSRunning},
			}},
		},
		{
			event.Event{Type: event.ETTransformPrint, Timestamp: 1609461100090, SessionID: runID, Payload: event.TransformMessage{Msg: "oh hai there"}},
			&State{ID: runID, StartTime: toTimePointer(1609460600090), Status: RSRunning, Steps: []*StepState{
				{Name: "setup", StartTime: toTimePointer(1609460700090), StopTime: toTimePointer(1609460900090), Duration: 200000, Status: RSSucceeded},
				{Name: "download", StartTime: toTimePointer(1609461000090), Status: RSRunning, Output: []event.Event{
					{Type: event.ETTransformPrint, Timestamp: 1609461100090, SessionID: runID, Payload: event.TransformMessage{Msg: "oh hai there"}},
				}},
			}},
		},
		// {
		// 	event.Event{ Type: event.ETHttpRequestStart, Timestamp: 1609461200090, SessionID: runID, Payload: {"id": runID, "downloadSize": 230409, "method": "Gevent.ET", "url": "https://registy.qri.cloud" }},
		// 	&State{},
		// {
		// {
		// event.Event{ Type: event.ETHttpRequestStop, Timestamp: 1609461300090, SessionID: runID, Payload: {"size": 230409, "method": "Gevent.ET", "url": "https://registy.qri.cloud" }},
		// &State{},
		// },
		{
			event.Event{Type: event.ETTransformStepStop, Timestamp: 1609461400090, SessionID: runID, Payload: event.TransformStepLifecycle{Name: "download", Status: "succeeded"}},
			&State{ID: runID, StartTime: toTimePointer(1609460600090), Status: RSRunning, Steps: []*StepState{
				{Name: "setup", StartTime: toTimePointer(1609460700090), StopTime: toTimePointer(1609460900090), Duration: 200000, Status: RSSucceeded},
				{Name: "download", StartTime: toTimePointer(1609461000090), StopTime: toTimePointer(1609461400090), Duration: 400000, Status: RSSucceeded, Output: []event.Event{
					{Type: event.ETTransformPrint, Timestamp: 1609461100090, SessionID: runID, Payload: event.TransformMessage{Msg: "oh hai there"}},
				}},
			}},
		},
		{
			event.Event{Type: event.ETTransformStepStart, Timestamp: 1609461500090, SessionID: runID, Payload: event.TransformStepLifecycle{Name: "transform"}},
			&State{ID: runID, StartTime: toTimePointer(1609460600090), Status: RSRunning, Steps: []*StepState{
				{Name: "setup", StartTime: toTimePointer(1609460700090), StopTime: toTimePointer(1609460900090), Duration: 200000, Status: RSSucceeded},
				{Name: "download", StartTime: toTimePointer(1609461000090), StopTime: toTimePointer(1609461400090), Duration: 400000, Status: RSSucceeded, Output: []event.Event{
					{Type: event.ETTransformPrint, Timestamp: 1609461100090, SessionID: runID, Payload: event.TransformMessage{Msg: "oh hai there"}},
				}},
				{Name: "transform", StartTime: toTimePointer(1609461500090), Status: RSRunning},
			}},
		},
		{
			event.Event{Type: event.ETTransformStepStop, Timestamp: 1609461600090, SessionID: runID, Payload: event.TransformStepLifecycle{Name: "transform", Status: "succeeded"}},
			&State{ID: runID, StartTime: toTimePointer(1609460600090), Status: RSRunning, Steps: []*StepState{
				{Name: "setup", StartTime: toTimePointer(1609460700090), StopTime: toTimePointer(1609460900090), Duration: 200000, Status: RSSucceeded},
				{Name: "download", StartTime: toTimePointer(1609461000090), StopTime: toTimePointer(1609461400090), Duration: 400000, Status: RSSucceeded, Output: []event.Event{
					{Type: event.ETTransformPrint, Timestamp: 1609461100090, SessionID: runID, Payload: event.TransformMessage{Msg: "oh hai there"}},
				}},
				{Name: "transform", StartTime: toTimePointer(1609461500090), StopTime: toTimePointer(1609461600090), Duration: 100000, Status: RSSucceeded},
			}},
		},
		{
			event.Event{Type: event.ETTransformStop, Timestamp: 1609461900090, SessionID: runID, Payload: event.TransformLifecycle{Status: "failed"}},
			&State{ID: runID, StartTime: toTimePointer(1609460600090), StopTime: toTimePointer(1609461900090), Duration: 1300000, Status: RSFailed, Steps: []*StepState{
				{Name: "setup", StartTime: toTimePointer(1609460700090), StopTime: toTimePointer(1609460900090), Duration: 200000, Status: RSSucceeded},
				{Name: "download", StartTime: toTimePointer(1609461000090), StopTime: toTimePointer(1609461400090), Duration: 400000, Status: RSSucceeded, Output: []event.Event{
					{Type: event.ETTransformPrint, Timestamp: 1609461100090, SessionID: runID, Payload: event.TransformMessage{Msg: "oh hai there"}},
				}},
				{Name: "transform", StartTime: toTimePointer(1609461500090), StopTime: toTimePointer(1609461600090), Duration: 100000, Status: RSSucceeded},
			}},
		},
	}

	for i, s := range states {
		t.Run(fmt.Sprintf("after_event_%d", i), func(t *testing.T) {
			got := &State{ID: runID}
			for j := 0; j <= i; j++ {
				if err := got.AddTransformEvent(states[j].e); err != nil {
					t.Fatal(err)
				}
			}

			if diff := cmp.Diff(s.r, got); diff != "" {
				t.Errorf("result mismatch. (-want +got):\n%s", diff)
			}
		})
	}
}
