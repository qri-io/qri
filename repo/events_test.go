package repo

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"

	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// compareEventSlices confirms two slices of events are equal
func compareEventSlices(a, b MemEventLog) error {
	if len(a) != len(b) {
		return fmt.Errorf("length mismatch: %d != %d", len(a), len(b))
	}

	for i, ae := range a {
		be := b[i]
		if err := compareEvents(ae, be); err != nil {
			return fmt.Errorf("event index %d error: %s", i, err.Error())
		}
	}

	return nil
}

// compareEvents checks to see if all elements of an event are the same,
// returning a rich error message on mismatch
func compareEvents(a, b *Event) error {
	if a.Type != b.Type {
		return fmt.Errorf("type mismatch. %s != %s", a.Type, b.Type)
	}
	if !a.Time.Equal(b.Time) {
		return fmt.Errorf("timestamp mismatch. %s != %s", a.Time, b.Time)
	}
	if !a.Ref.Equal(b.Ref) {
		return fmt.Errorf("ref mismatch. %s != %s", a.Ref, b.Ref)
	}
	if a.PeerID != b.PeerID {
		return fmt.Errorf("peerID mismatch: %s != %s", a.PeerID, b.PeerID)
	}
	if !reflect.DeepEqual(a.Params, b.Params) {
		return fmt.Errorf("params mismatch")
	}
	return nil
}

func TestEventsSince(t *testing.T) {
	tests := []struct {
		memEventLog MemEventLog
		time        time.Time
		expected    MemEventLog
	}{
		{MemEventLog{}, time.Unix(0, 0), MemEventLog{}},
		{MemEventLog{}, time.Unix(5, 0), MemEventLog{}},
		{MemEventLog{
			&Event{Type: ETDsAdded, Time: time.Unix(10, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(20, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(40, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(50, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(60, 0)},
		}, time.Unix(0, 0), MemEventLog{
			&Event{Type: ETDsDeleted, Time: time.Unix(60, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(50, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(40, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(20, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(10, 0)},
		}},
		{MemEventLog{
			&Event{Type: ETDsAdded, Time: time.Unix(5, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(10, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(15, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(20, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(25, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
		}, time.Unix(15, 0), MemEventLog{
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(25, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(20, 0)},
		}},
		{MemEventLog{
			&Event{Type: ETDsAdded, Time: time.Unix(5, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(10, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(15, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(20, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(25, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
		}, time.Unix(30, 0), MemEventLog{}},
	}

	for i, test := range tests {
		results, err := test.memEventLog.EventsSince(test.time)
		if err != nil {
			t.Errorf("Case %d had an EventsSlice error: %s", i, err.Error())
		}

		err = compareEventSlices(results, test.expected)
		if err != nil {
			t.Errorf("Case %d unexpected error: %s", i, err.Error())
		}
	}

}

func TestEvents(t *testing.T) {
	tests := []struct {
		memEventLog MemEventLog
		limit       int
		offset      int
		expected    MemEventLog
	}{
		{MemEventLog{}, 0, 0, MemEventLog{}},
		{MemEventLog{}, 1, 0, MemEventLog{}},
		{MemEventLog{}, 0, 1, MemEventLog{}},
		{MemEventLog{
			&Event{Type: ETDsAdded, Time: time.Unix(5, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(10, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(15, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(20, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(25, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
		}, 0, 0, MemEventLog{}},
		{MemEventLog{
			&Event{Type: ETDsAdded, Time: time.Unix(5, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(10, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(15, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(20, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(25, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
		}, 2, 0, MemEventLog{
			&Event{Type: ETDsAdded, Time: time.Unix(5, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(10, 0)},
		}},
		{MemEventLog{
			&Event{Type: ETDsAdded, Time: time.Unix(5, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(10, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(15, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(20, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(25, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
		}, 2, 2, MemEventLog{
			&Event{Type: ETDsDeleted, Time: time.Unix(15, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(20, 0)},
		}},
		{MemEventLog{
			&Event{Type: ETDsAdded, Time: time.Unix(5, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(10, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(15, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(20, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(25, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
		}, 2, 5, MemEventLog{
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
		}},
		{MemEventLog{
			&Event{Type: ETDsAdded, Time: time.Unix(5, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(10, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(15, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(20, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(25, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
		}, 10, 0, MemEventLog{
			&Event{Type: ETDsAdded, Time: time.Unix(5, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(10, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(15, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(20, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(25, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
		}},
		{MemEventLog{
			&Event{Type: ETDsAdded, Time: time.Unix(5, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(10, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(15, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(20, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(25, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
		}, 0, 10, MemEventLog{}},
		{MemEventLog{
			&Event{Type: ETDsAdded, Time: time.Unix(5, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(10, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(15, 0)},
			&Event{Type: ETDsAdded, Time: time.Unix(20, 0)},
			&Event{Type: ETDsCreated, Time: time.Unix(25, 0)},
			&Event{Type: ETDsDeleted, Time: time.Unix(30, 0)},
		}, 0, 2, MemEventLog{}},
	}

	for i, test := range tests {
		results, err := test.memEventLog.Events(test.limit, test.offset)
		if err != nil {
			t.Errorf("Case %d had an Events err: %s", i, err.Error())
		}

		err = compareEventSlices(results, test.expected)
		if err != nil {
			t.Errorf("Case %d unexpected error: %s", i, err.Error())
		}
	}
}

func TestLogEventDetails(t *testing.T) {
	type data struct {
		event      EventType
		when       int64
		peerID     peer.ID
		datasetRef DatasetRef
		params     interface{}
	}

	tests := []struct {
		logs     []data
		expected int
	}{
		{[]data{}, 0},
		{[]data{
			data{ETDsAdded, rand.Int63n(math.MaxInt32), peer.ID("ID"), DatasetRef{}, nil},
			data{ETDsCreated, rand.Int63n(math.MaxInt32), peer.ID("ID"), DatasetRef{}, nil},
			data{ETDsDeleted, rand.Int63n(math.MaxInt32), peer.ID("ID"), DatasetRef{}, nil},
			data{ETDsRenamed, rand.Int63n(math.MaxInt32), peer.ID("ID"), DatasetRef{}, nil},
			data{ETDsPinned, rand.Int63n(math.MaxInt32), peer.ID("ID"), DatasetRef{}, nil},
			data{ETDsUnpinned, rand.Int63n(math.MaxInt32), peer.ID("ID"), DatasetRef{}, nil},
			data{ETTransformExecuted, rand.Int63n(math.MaxInt32), peer.ID("ID"), DatasetRef{}, nil},
		}, 7},
	}

	for i, test := range tests {
		memEventLog := MemEventLog{}

		for _, log := range test.logs {
			err := memEventLog.LogEventDetails(log.event, log.when, log.peerID, log.datasetRef, log.params)
			if err != nil {
				t.Errorf("Case %d had a LogEventDetails error: %s", i, err.Error())
			}
		}

		if len(memEventLog) != test.expected {
			t.Errorf("Case %d expected the number of events to be %d, but got %d", i, test.expected, len(memEventLog))
		}

		for index := len(memEventLog); i > 1; i-- {
			if memEventLog[index-1].Time.After(memEventLog[index].Time) {
				for index2, event := range memEventLog {
					t.Logf("Logs Not sorted:\n")
					t.Logf("index: %d, time: %s\n", index2, event.Time)
				}
				t.Errorf("Case %d expected the logs to be sorted, but they were not", i)
			}
		}
	}
}

func TestLogEvent(t *testing.T) {
	tests := []struct {
		events   []EventType
		expected int
	}{
		{[]EventType{}, 0},
		{[]EventType{ETDsAdded}, 1},
		{[]EventType{ETDsAdded, ETDsCreated, ETDsDeleted}, 3},
		{[]EventType{ETDsAdded, ETDsCreated, ETDsDeleted, ETDsRenamed, ETDsPinned, ETDsUnpinned, ETTransformExecuted}, 7},
	}

	for i, test := range tests {
		memEventLog := MemEventLog{}
		datasetRef := DatasetRef{}

		for _, event := range test.events {
			err := memEventLog.LogEvent(event, datasetRef)
			if err != nil {
				t.Errorf("Case %d had a LogEvent error: %s", i, err.Error())
			}
		}

		if len(memEventLog) != test.expected {
			t.Errorf("Case %d expected the number of events to be %d, but got %d", i, test.expected, len(memEventLog))
		}
	}

}
