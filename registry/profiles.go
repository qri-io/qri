package registry

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	// nowFunc is an internal function for getting timestamps
	nowFunc = func() time.Time { return time.Now() }
)

// Profiles is the interface for working with a set of *Profile's
// Register, Deregister, Load, Len, Range, and SortedRange should be
// considered safe to hook up to public http endpoints, whereas
// Delete & Store should only be exposed in administrative contexts
// users should prefer using RegisterProfile & DegristerProfile for
// dataset manipulation operations
type Profiles interface {
	// Len returns the number of records in the set
	Len() int
	// Load fetches a profile from the list by key
	Load(key string) (value *Profile, ok bool)
	// Range calls an iteration fuction on each element in the map until
	// the end of the list is reached or iter returns true
	Range(iter func(key string, p *Profile) (brk bool))
	// SortedRange is like range but with deterministic key ordering
	SortedRange(iter func(key string, p *Profile) (brk bool))

	// Store adds an entry, bypassing the register process
	// store is only exported for administrative use cases.
	// most of the time callers should use Register instead
	Store(key string, value *Profile)
	// Delete removes a record from the set at key
	// Delete is only exported for administrative use cases.
	// most of the time callers should use Deregister instead
	Delete(key string)
}

// RegisterProfile adds a profile to the list if it's valid and the desired handle isn't taken
func RegisterProfile(store Profiles, p *Profile) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if err := p.Verify(); err != nil {
		return err
	}

	if pro, ok := store.Load(p.Username); ok {
		// if peer is registring a name they already own, we're good
		if pro.ProfileID == p.ProfileID {
			return nil
		}
		return fmt.Errorf("username '%s' is taken", p.Username)
	}

	prev := ""
	store.Range(func(key string, profile *Profile) bool {
		if profile.ProfileID == p.ProfileID {
			prev = key
			return true
		}
		return false
	})

	if prev != "" {
		store.Delete(prev)
	}

	store.Store(p.Username, &Profile{
		Username:  p.Username,
		Created:   nowFunc(),
		ProfileID: p.ProfileID,
		PublicKey: p.PublicKey,
	})
	return nil
}

// DeregisterProfile removes a profile from the registry if it exists
// confirming the user has the authority to do so
func DeregisterProfile(store Profiles, p *Profile) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if err := p.Verify(); err != nil {
		return err
	}

	store.Delete(p.Username)
	return nil
}

// MemProfiles is a map of profile data safe for concurrent use
// heavily inspired by sync.Map
type MemProfiles struct {
	sync.RWMutex
	internal map[string]*Profile
}

// NewMemProfiles allocates a new *MemProfiles map
func NewMemProfiles() *MemProfiles {
	return &MemProfiles{
		internal: make(map[string]*Profile),
	}
}

// Len returns the number of records in the map
func (ps *MemProfiles) Len() int {
	return len(ps.internal)
}

// Load fetches a profile from the list by key
func (ps *MemProfiles) Load(key string) (value *Profile, ok bool) {
	ps.RLock()
	result, ok := ps.internal[key]
	ps.RUnlock()
	return result, ok
}

// Range calls an iteration fuction on each element in the map until
// the end of the list is reached or iter returns true
func (ps *MemProfiles) Range(iter func(key string, p *Profile) (brk bool)) {
	ps.RLock()
	defer ps.RUnlock()
	for key, p := range ps.internal {
		if iter(key, p) {
			break
		}
	}
}

// SortedRange is like range but with deterministic key ordering
func (ps *MemProfiles) SortedRange(iter func(key string, p *Profile) (brk bool)) {
	ps.RLock()
	defer ps.RUnlock()
	keys := make([]string, len(ps.internal))
	i := 0
	for key := range ps.internal {
		keys[i] = key
		i++
	}
	sort.StringSlice(keys).Sort()
	for _, key := range keys {
		if iter(key, ps.internal[key]) {
			break
		}
	}
}

// Delete removes a record from MemProfiles at key
func (ps *MemProfiles) Delete(key string) {
	ps.Lock()
	delete(ps.internal, key)
	ps.Unlock()
}

// Store adds an entry
func (ps *MemProfiles) Store(key string, value *Profile) {
	ps.Lock()
	ps.internal[key] = value
	ps.Unlock()
}
