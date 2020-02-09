package dscache

import (
	"github.com/qri-io/qri/dsref"
)

// Builder can be used to construct a dscache directly
type Builder struct {
	filename string
	users    []userProfilePair
	infos    []*dsInfo
}

// NewBuilder returns a new dscache builder
func NewBuilder() *Builder {
	return &Builder{
		users: []userProfilePair{},
		infos: []*dsInfo{},
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
func (b *Builder) AddDsVersionInfo(initID string, ver dsref.VersionInfo) {
	b.infos = append(b.infos, &dsInfo{
		InitID:        initID,
		ProfileID:     ver.ProfileID,
		TopIndex:      -1,
		CursorIndex:   -1,
		PrettyName:    ver.Name,
		Published:     ver.Published,
		Foreign:       ver.Foreign,
		MetaTitle:     ver.MetaTitle,
		ThemeList:     ver.ThemeList,
		BodySize:      int64(ver.BodySize),
		BodyRows:      ver.BodyRows,
		BodyFormat:    ver.BodyFormat,
		NumErrors:     ver.NumErrors,
		CommitTime:    ver.CommitTime,
		CommitTitle:   ver.CommitTitle,
		CommitMessage: ver.CommitMessage,
		NumVersions:   ver.NumVersions,
		HeadRef:       ver.Path,
		FSIPath:       ver.FSIPath,
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
