package resolver

import (
	"github.com/qri-io/qri/dsref"
)

// Resolver resolves identifiers into info about datasets
type Resolver interface {
	GetInfo(initID string) *dsref.VersionInfo
	GetInfoByDsref(dr dsref.Ref) *dsref.VersionInfo
}
