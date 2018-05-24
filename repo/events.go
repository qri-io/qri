package repo

import (
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	"sort"
	"time"
)

// EventLog keeps logs
type EventLog interface {
	LogEvent(t EventType, ref DatasetRef) error
	Events(limit, offset int) ([]*Event, error)
	EventsSince(time.Time) ([]*Event, error)
}

// Event is a list of details for logging a query
type Event struct {
	Time   time.Time
	Type   EventType
	Ref    DatasetRef
	PeerID peer.ID
	Params interface{}
}

// EventType classifies types of events that can be logged
type EventType string

const (
	// ETDsCreated represents a peer creating a dataset (either updating an existing or a new dataset)
	ETDsCreated = EventType("ds_created")
	// ETDsDeleted represents destroying a dataset. Peers should respect this and remove locally as well
	ETDsDeleted = EventType("ds_deleted")
	// ETDsRenamed represents changing a dataset's name. Peers should update their refstore
	ETDsRenamed = EventType("ds_renamed")
	// ETDsPinned represents a peer pinning a dataset to their local storage
	ETDsPinned = EventType("ds_pinned")
	// ETDsUnpinned represents a peer unpinnning a dataset from local storage
	ETDsUnpinned = EventType("ds_unpinned")
	// ETDsAdded represents adding a reference to another peer's dataset to their node
	ETDsAdded = EventType("ds_added")
	// ETTransformExecuted represents running a transformation
	ETTransformExecuted = EventType("tf_executed")
)

// MemEventLog is an in-memory implementation of the
// EventLog interface
type MemEventLog []*Event

// LogEvent adds a query entry to the store
func (log *MemEventLog) LogEvent(t EventType, ref DatasetRef) error {
	e := &Event{
		Time: time.Now(),
		Type: t,
		Ref:  ref,
	}
	logs := append([]*Event{e}, *log...)
	sort.Slice(logs, func(i, j int) bool { return logs[i].Time.After(logs[j].Time) })
	*log = logs
	return nil
}

// LogEventDetails adds an entry to the log
// TODO: Update LogEvent to work like this, update callers.
func (log *MemEventLog) LogEventDetails(t EventType, when int64, peerID peer.ID, ref DatasetRef, params interface{}) error {
	e := &Event{
		Time:   time.Unix(when, 0),
		Type:   t,
		Ref:    ref,
		PeerID: peerID,
		Params: params,
	}
	logs := append([]*Event{e}, *log...)
	sort.Slice(logs, func(i, j int) bool { return logs[i].Time.After(logs[j].Time) })
	*log = logs
	return nil
}

// Events grabs a set of Events from the store
func (log MemEventLog) Events(limit, offset int) ([]*Event, error) {
	if offset > len(log) {
		offset = len(log)
	}
	stop := limit + offset
	if stop > len(log) {
		stop = len(log)
	}

	return log[offset:stop], nil
}

// EventsSince produces a slice of all events since a given time
func (log MemEventLog) EventsSince(t time.Time) ([]*Event, error) {
	events := make([]*Event, 0, len(log))
	for _, e := range log {
		if e.Time.After(t) {
			events = append([]*Event{e}, events...)
		}
	}

	return events, nil
}
