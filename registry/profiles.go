package registry

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	// nowFunc is an ps function for getting timestamps
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
	Len() (int, error)
	// Load fetches a profile from the list by key
	Load(key string) (value *Profile, err error)
	// Range calls an iteration fuction on each element in the map until
	// the end of the list is reached or iter returns true
	Range(iter func(key string, p *Profile) (kontinue bool, err error)) error
	// SortedRange is like range but with deterministic key ordering
	SortedRange(iter func(key string, p *Profile) (kontinue bool, err error)) error

	// Create adds an entry, bypassing the register process
	// store is only exported for administrative use cases.
	// most of the time callers should use Register instead
	Create(key string, value *Profile) error
	// Update modifies an existing profile
	Update(key string, value *Profile) error
	// Delete removes a profile from the set at key
	// Delete is only exported for administrative use cases.
	// most of the time callers should use Deregister instead
	Delete(key string) error
}

// RegisterProfile adds a profile to the list if it's valid and the desired handle isn't taken
func RegisterProfile(store Profiles, p *Profile) (err error) {
	if err = p.Validate(); err != nil {
		return err
	}
	if err = p.Verify(); err != nil {
		return err
	}

	pro, err := store.Load(p.Username)
	if err == nil {
		// if peer is registring a name they already own, we're good
		if pro.ProfileID == p.ProfileID {
			return nil
		}
		return fmt.Errorf("username '%s' is taken", p.Username)
	}

	prev := ""
	store.Range(func(key string, profile *Profile) (bool, error) {
		if profile.ProfileID == p.ProfileID {
			prev = key
			return true, nil
		}
		return false, nil
	})

	if prev != "" {
		if err = store.Delete(prev); err != nil {
			return err
		}
	}

	p.Created = time.Now()
	return store.Create(p.Username, p)
}

// UpdateProfile alters profile data
func UpdateProfile(store Profiles, p *Profile) (err error) {
	if err = p.Validate(); err != nil {
		return err
	}
	if err = p.Verify(); err != nil {
		return err
	}

	return store.Update(p.Username, p)
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
	ps map[string]*Profile
}

var _ Profiles = (*MemProfiles)(nil)

// NewMemProfiles allocates a new *MemProfiles map
func NewMemProfiles() *MemProfiles {
	return &MemProfiles{
		ps: make(map[string]*Profile),
	}
}

// Len returns the number of records in the map
func (ps *MemProfiles) Len() (int, error) {
	return len(ps.ps), nil
}

// Load fetches a profile from the list by key
func (ps *MemProfiles) Load(key string) (value *Profile, err error) {
	ps.RLock()
	result, ok := ps.ps[key]
	ps.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}

	return result, nil
}

// Range calls an iteration fuction on each element in the map until
// the end of the list is reached or iter returns true
func (ps *MemProfiles) Range(iter func(key string, p *Profile) (kontinue bool, err error)) (err error) {
	ps.RLock()
	defer ps.RUnlock()
	var kontinue bool

	for key, p := range ps.ps {
		kontinue, err = iter(key, p)
		if err != nil {
			return err
		}
		if !kontinue {
			break
		}
	}

	return nil
}

// SortedRange is like range but with deterministic key ordering
func (ps *MemProfiles) SortedRange(iter func(key string, p *Profile) (kontinue bool, err error)) (err error) {
	ps.RLock()
	defer ps.RUnlock()
	keys := make([]string, len(ps.ps))
	i := 0
	for key := range ps.ps {
		keys[i] = key
		i++
	}
	sort.StringSlice(keys).Sort()

	var kontinue bool
	for _, key := range keys {
		kontinue, err = iter(key, ps.ps[key])
		if err != nil {
			return err
		}
		if !kontinue {
			break
		}
	}
	return nil
}

// Delete removes a record from MemProfiles at key
func (ps *MemProfiles) Delete(key string) error {
	ps.Lock()
	delete(ps.ps, key)
	ps.Unlock()
	return nil
}

// Create adds a profile
func (ps *MemProfiles) Create(key string, value *Profile) error {
	ps.Lock()
	ps.ps[key] = value
	ps.Unlock()
	return nil
}

// Update modifies an existing profile
func (ps *MemProfiles) Update(key string, value *Profile) error {
	ps.Lock()
	ps.ps[key] = value
	ps.Unlock()
	return nil
}
