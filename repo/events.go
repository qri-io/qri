package repo

import (
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
	Time time.Time
	Type EventType
	Ref  DatasetRef
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
	sort.Slice(logs, func(i, j int) bool { return logs[i].Time.Before(logs[j].Time) })
	*log = logs
	return nil
}

// Event fills a partial Event with all details from the store
// func (log *MemEventLog) Event(q *Event) (*Event, error) {
// 	for _, item := range *log {
// 		if item.DatasetPath.Equal(q.DatasetPath) ||
// 			item.Event == q.Event ||
// 			item.Time.Equal(q.Time) ||
// 			item.Key.Equal(q.Key) {
// 			return item, nil
// 		}
// 	}
// 	return nil, ErrNotFound
// }

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
