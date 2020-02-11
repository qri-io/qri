package reporef

import (
	"strings"

	"github.com/qri-io/qri/dsref"
)

// ConvertToVersionInfo converts an old style DatasetRef to the newly preferred dsref.VersionInfo
func ConvertToVersionInfo(r *DatasetRef) dsref.VersionInfo {
	build := dsref.VersionInfo{
		Username:  r.Peername,
		ProfileID: r.ProfileID.String(),
		Name:      r.Name,
		Path:      r.Path,
	}
	ds := r.Dataset
	if ds != nil && ds.Meta != nil {
		if ds.Meta.Title != "" {
			build.MetaTitle = ds.Meta.Title
		}
		if ds.Meta.Theme != nil {
			build.ThemeList = strings.Join(ds.Meta.Theme, ",")
		}
	}
	if r.FSIPath != "" {
		build.FSIPath = r.FSIPath
	}
	if r.Foreign {
		build.Foreign = true
	}
	if ds != nil && ds.Structure != nil {
		build.BodySize = ds.Structure.Length
		build.BodyRows = ds.Structure.Entries
		build.NumErrors = ds.Structure.ErrCount
		build.BodyFormat = ds.Structure.Format
	}
	if ds != nil && ds.Commit != nil {
		build.CommitTime = ds.Commit.Timestamp
		build.CommitTitle = ds.Commit.Title
		build.CommitMessage = ds.Commit.Message
	}
	if ds != nil {
		build.NumVersions = ds.NumVersions
	}
	build.Published = r.Published
	build.Foreign = r.Foreign
	// NOTE: InitID is not set when converting from reporef.Dataset
	return build
}

// ConvertToDsref is a shim function to transition from a reporef.DatasetRef to a
// dsref.Ref while we experiment with dsref as the home of name parsing
func ConvertToDsref(ref DatasetRef) dsref.Ref {
	return dsref.Ref{
		Username:  ref.Peername,
		Name:      ref.Name,
		ProfileID: ref.ProfileID.String(),
		Path:      ref.Path,
	}
}
