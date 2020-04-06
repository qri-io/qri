package resolver

import (
	"fmt"

	"github.com/qri-io/qri/dsref"
)

var _ Resolver = (*MemResolver)(nil)

// MemResolver holds maps that can do a cheap version of dataset resolution, for tests
type MemResolver struct {
	RefMap map[string]string
	IDMap  map[string]dsref.VersionInfo
}

// NewMemResolver returns a new MemResolver
func NewMemResolver() *MemResolver {
	return &MemResolver{
		RefMap: make(map[string]string),
		IDMap:  make(map[string]dsref.VersionInfo),
	}
}

// Put adds a VersionInfo to the resolver
func (m *MemResolver) Put(info dsref.VersionInfo) {
	refStr := fmt.Sprintf("%s/%s", info.Username, info.Name)
	initID := info.InitID
	m.RefMap[refStr] = initID
	m.IDMap[initID] = info
}

// GetInfo returns a VersionInfo by initID, or nil if not found
func (m *MemResolver) GetInfo(initID string) *dsref.VersionInfo {
	if info, ok := m.IDMap[initID]; ok {
		return &info
	}
	return nil
}

// GetInfoByDsref returns a VersionInfo by dsref, or nil if not found
func (m *MemResolver) GetInfoByDsref(dr dsref.Ref) *dsref.VersionInfo {
	refStr := fmt.Sprintf("%s/%s", dr.Username, dr.Name)
	if initID, ok := m.RefMap[refStr]; ok {
		return m.GetInfo(initID)
	}
	return nil
}
