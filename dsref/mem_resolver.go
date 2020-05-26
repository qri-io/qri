package dsref

import (
	"context"
	"fmt"
)

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
		return "", ErrRefNotFound
	}

	id := m.RefMap[ref.Alias()]
	resolved, ok := m.IDMap[id]
	if !ok {
		return "", ErrRefNotFound
	}

	ref.InitID = id
	if ref.Path == "" {
		ref.Path = resolved.Path
	}

	return "", nil
}
