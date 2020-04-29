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

	// Handle the "me" convenience shortcut
	if ref.Username == "me" {
		ref.Username = m.Username
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
