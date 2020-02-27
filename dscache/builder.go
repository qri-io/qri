package dscache

import (
	"github.com/qri-io/qri/dsref"
)

// Builder can be used to construct a dscache directly
type Builder struct {
	filename string
	users    []userProfilePair
	infos    []*entryInfo
}

// NewBuilder returns a new dscache builder
func NewBuilder() *Builder {
	return &Builder{
		users: []userProfilePair{},
		infos: []*entryInfo{},
	}
}

// SetFilename sets the filename for the dscache
func (b *Builder) SetFilename(filename string) {
	b.filename = filename
}

// AddUser adds a user to the dscache
func (b *Builder) AddUser(username, profileID string) {
	b.users = append(b.users, userProfilePair{Username: username, ProfileID: profileID})
}

// AddDsVersionInfo adds a versionInfo to the dscache
func (b *Builder) AddDsVersionInfo(ver dsref.VersionInfo) {
	b.AddDsVersionInfoWithIndexes(ver, -1, -1)
}

// AddDsVersionInfoWithIndexes adds a versionInfo with indexes to the dscache
func (b *Builder) AddDsVersionInfoWithIndexes(ver dsref.VersionInfo, topIndex, cursorIndex int) {
	b.infos = append(b.infos, &entryInfo{
		VersionInfo: ver,
		TopIndex:    topIndex,
		CursorIndex: cursorIndex,
	})
}

// Build returns the built Dscache
func (b *Builder) Build() *Dscache {
	cache := buildDscacheFlatbuffer(b.users, b.infos)
	if b.filename != "" {
		cache.Filename = b.filename
	}
	return cache
}
