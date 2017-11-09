package analytics

import (
	"github.com/ipfs/go-datastore/query"
	"time"
)

// Analytics is the interface for collecting and storing
// analytics events
type Analytics interface {
	Query(query.Query) (query.Results, error)
	Track(event string, props map[string]interface{}) error
}

// Event is a tracked analytics event
type Event struct {
	Name    string
	Created time.Time
	Props   map[string]interface{}
}

// Memstore is a basic implementation of
// the Analytics interface
type Memstore []*Event

// Track records an event
func (ms *Memstore) Track(event string, props map[string]interface{}) error {
	*ms = append(*ms, &Event{Name: event, Created: time.Now(), Props: props})
	return nil
}

// Query makes the memstore Queryable
func (ms Memstore) Query(q query.Query) (query.Results, error) {
	re := make([]query.Entry, 0, len(ms))
	for _, v := range ms {
		re = append(re, query.Entry{Key: v.Name, Value: v})
	}
	r := query.ResultsWithEntries(q, re)
	r = query.NaiveQueryApply(q, r)
	return r, nil
}
