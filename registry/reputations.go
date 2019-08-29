package registry

import (
	"sort"
	"sync"
)

// Reputations is the interface for working with a set of *Reputation's
// Add, Load, Len, Range, and SortedRange should be
// considered safe to hook up to public http endpoints, whereas
// Delete & Store should only be exposed in administrative contexts
type Reputations interface {
	// Len returns the number of records in the set
	Len() int
	// Load fetches a profile from the list by key
	Load(key string) (value *Reputation, ok bool)
	// Range calls an iteration fuction on each element in the map until
	// the end of the list is reached or iter returns true
	Range(iter func(key string, r *Reputation) (brk bool))
	// SortedRange is like range but with deterministic key ordering
	SortedRange(iter func(key string, r *Reputation) (brk bool))
	// Add adds the Reputation to the list of reputations
	Add(r *Reputation) error

	// Store adds an entry, bypassing the register process
	// store is only exported for administrative use cases.
	// most of the time callers should use Register instead
	Store(key string, value *Reputation)
	// Delete removes a record from the set at key
	// Delete is only exported for administrative use cases.
	// most of the time callers should use Deregister instead
	Delete(key string)
}

// MemReputations is a map of reputation data safe for concurrent use
// heavily inspired by sync.Map
type MemReputations struct {
	sync.RWMutex
	internal map[string]*Reputation
}

// NewMemReputations allocates a new *MemReputations map
func NewMemReputations() *MemReputations {
	return &MemReputations{
		internal: make(map[string]*Reputation),
	}
}

// Add adds the reputation to the map of reputations
// it Validates the reputation before adding it
func (rs *MemReputations) Add(r *Reputation) error {
	err := r.Validate()
	if err != nil {
		return err
	}
	rs.Store(r.ProfileID, r)
	return nil
}

// Len returns the number of records in the map
func (rs *MemReputations) Len() int {
	return len(rs.internal)
}

// Load fetches a reputation from the list by key
func (rs *MemReputations) Load(key string) (value *Reputation, ok bool) {
	rs.RLock()
	result, ok := rs.internal[key]
	rs.RUnlock()
	return result, ok
}

// Range calls an iteration fuction on each element in the map until
// the end of the list is reached or iter returns true
func (rs *MemReputations) Range(iter func(key string, r *Reputation) (brk bool)) {
	rs.RLock()
	defer rs.RUnlock()
	for key, r := range rs.internal {
		if iter(key, r) {
			break
		}
	}
}

// SortedRange is like range but with deterministic key ordering
func (rs *MemReputations) SortedRange(iter func(key string, r *Reputation) (brk bool)) {
	rs.RLock()
	defer rs.RUnlock()
	keys := make([]string, len(rs.internal))
	i := 0
	for key := range rs.internal {
		keys[i] = key
		i++
	}
	sort.StringSlice(keys).Sort()
	for _, key := range keys {
		if iter(key, rs.internal[key]) {
			break
		}
	}
}

// Delete removes a record from MemReputations at key
func (rs *MemReputations) Delete(key string) {
	rs.Lock()
	delete(rs.internal, key)
	rs.Unlock()
}

// Store adds an entry
func (rs *MemReputations) Store(key string, value *Reputation) {
	rs.Lock()
	rs.internal[key] = value
	rs.Unlock()
}
