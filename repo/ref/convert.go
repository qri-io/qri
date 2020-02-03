package reporef

import (
	"strings"

	"github.com/qri-io/qri/dsref"
)

// ConvertToDetailedRef converts an old style DatasetRef to the newly preferred dsref.DetailedRef
func ConvertToDetailedRef(r *DatasetRef) dsref.DetailedRef {
	build := dsref.DetailedRef{
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
	}
	if ds != nil && ds.Commit != nil {
		build.CommitTime = ds.Commit.Timestamp
	}
	if ds != nil {
		build.NumVersions = ds.NumVersions
	}
	return build
}
