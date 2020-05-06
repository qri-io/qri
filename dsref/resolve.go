package dsref

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrNotFound must be returned by a ref resolver that cannot resolve a given
	// reference
	ErrNotFound = errors.New("reference not found")
)

// Resolver finds the identifier and HEAD path for a dataset reference
type Resolver interface {
	// ResolveRef uses ref as an outParam, setting ref.ID and ref.Path on success
	// some implementations of name resolution may make network calls
	// the returned "source" value should be either an empty string, indicating
	// the source was resolved locally, or the multiaddress of the network
	// address that performed the resolution
	ResolveRef(ctx context.Context, ref *Ref) (source string, err error)
}

// MemResolver holds maps that can do a cheap version of dataset resolution,
// for tests
type MemResolver struct {
	Username string
	RefMap   map[string]string
	IDMap    map[string]VersionInfo
}

// assert at compile time that MemResolver is a Resolver
var _ Resolver = (*MemResolver)(nil)

// NewMemResolver returns a new MemResolver
func NewMemResolver(username string) *MemResolver {
	return &MemResolver{
		Username: username,
		RefMap:   make(map[string]string),
		IDMap:    make(map[string]VersionInfo),
	}
}

// Put adds a VersionInfo to the resolver
func (m *MemResolver) Put(info VersionInfo) {
	refStr := fmt.Sprintf("%s/%s", info.Username, info.Name)
	initID := info.InitID
	m.RefMap[refStr] = initID
	m.IDMap[initID] = info
}

// GetInfo returns a VersionInfo by initID, or nil if not found
func (m *MemResolver) GetInfo(initID string) *VersionInfo {
	if info, ok := m.IDMap[initID]; ok {
		return &info
	}
	return nil
}

// ResolveRef finds the identifier & head path for a dataset reference
// implements resolve.NameResolver interface
func (m *MemResolver) ResolveRef(ctx context.Context, ref *Ref) (string, error) {
	if m == nil {
		return "", ErrNotFound
	}

	id := m.RefMap[ref.Alias()]
	resolved, ok := m.IDMap[id]
	if !ok {
		return "", ErrNotFound
	}

	ref.InitID = id
	if ref.Path == "" {
		ref.Path = resolved.Path
	}

	return "", nil
}

type parallelResolver []Resolver

func (rs parallelResolver) ResolveRef(ctx context.Context, ref *Ref) (string, error) {
	responses := make(chan struct {
		Ref    Ref
		Source string
	})
	errs := make(chan error)

	run := func(ctx context.Context, r Resolver) {
		if r == nil {
			errs <- ErrNotFound
			return
		}

		cpy := ref.Copy()
		source, err := r.ResolveRef(ctx, &cpy)
		if err != nil {
			errs <- err
			return
		}

		responses <- struct {
			Ref    Ref
			Source string
		}{cpy, source}
	}

	attempts := len(rs)
	for _, r := range rs {
		go run(ctx, r)
	}

	for {
		select {
		case res := <-responses:
			*ref = res.Ref
			return res.Source, nil
		case err := <-errs:
			attempts--
			if !errors.Is(err, ErrNotFound) {
				return "", err
			} else if attempts == 0 {
				return "", ErrNotFound
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// ParallelResolver composes multiple resolvers into one resolver that runs
// in parallel when called, returning the first valid response
func ParallelResolver(resolvers ...Resolver) Resolver {
	return parallelResolver(resolvers)
}
