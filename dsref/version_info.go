package dsref

import (
	"strings"
	"time"

	"github.com/qri-io/dataset"
)

// VersionInfo is an aggregation of fields from a dataset version for caching &
// listing purposes. VersionInfos are typically used when showing a list of
// datasets or a list of dataset versions ("qri list" and "qri log").
//
// VersionInfos can also describe dataset versions that are being created or
// failed to create. In these cases the calculated VersionInfo.Path value must
// always equal the empty string.
//
// If any fields are added to this struct, keep it in sync with:
//   dscache/def.fbs       dscache
//   dscache/fill_info.go  func fillInfoForDatasets
//   repo/ref/convert.go   func ConvertToVersionInfo
// If you are considering making major changes to VersionInfo, read this
// synopsis first:
//   https://github.com/qri-io/qri/pull/1641#issuecomment-778521313
type VersionInfo struct {
	//
	// Key as a stable identifier
	//
	// InitID is derived from the logbook for the dataset
	InitID string `json:"initID,omitempty"`
	//
	// Fields from dsref.Ref
	//
	// Username of dataset owner
	Username string `json:"username,omitempty"`
	// ProfileID of dataset owner
	ProfileID string `json:"profileID,omitempty"`
	// Unique name reference for this dataset
	Name string `json:"name,omitempty"`
	// Content-addressed path for this dataset
	Path string `json:"path,omitempty"`
	//
	// State about the dataset that can change
	//
	// If true, this dataset has published versions
	Published bool `json:"published,omitempty"`
	// If true, this reference doesn't exist locally. Only makes sense if path is set, as this
	// flag refers to specific versions, not to entire dataset histories.
	Foreign bool `json:"foreign,omitempty"`
	//
	// Meta fields
	//
	// Title from the meta structure
	MetaTitle string `json:"metaTitle,omitempty"`
	// List of themes from the meta structure, comma-separated list
	ThemeList string `json:"themeList,omitempty"`
	//
	// Structure fields
	//
	// Size of the body in bytes
	BodySize int `json:"bodySize,omitempty"`
	// Num of rows in the body
	BodyRows int `json:"bodyRows,omitempty"`
	// Format of the body, such as "csv" or "json"
	BodyFormat string `json:"bodyFormat,omitempty"`
	// Number of errors from the structure
	NumErrors int `json:"numErrors,omitempty"`
	//
	// Commit fields
	//
	// Timestamp field from the commit
	CommitTime time.Time `json:"commitTime,omitempty"`
	// Title field from the commit
	CommitTitle string `json:"commitTitle,omitempty"`
	// Message field from the commit
	CommitMessage string `json:"commitMessage,omitempty"`
	//
	// About the dataset's history and location
	//
	// Number of versions that the dataset has
	NumVersions int `json:"numVersions,omitempty"`
	// FSIPath is this dataset's link to the local filesystem if one exists
	FSIPath string `json:"fsiPath,omitempty"`
	//
	// Run Fields
	//
	// RunID is derived from from either the Commit.RunID, field or the runID of a
	// failed run. In the latter case the Path value will be empty
	RunID string `json:"runID,omitempty"`
	// RunStatus is a string version of the run.Status enumeration. This value
	// will always be one of:
	//    ""|"waiting"|"running"|"succeeded"|"failed"|"unchanged"|"skipped"
	// RunStatus is not stored on a dataset version, and instead must come from
	// either run state or a cache of run state
	// it's of type string to follow the "plain old data" pattern
	RunStatus string `json:"runStatus,omitempty"`
	// RunDuration is how long the run took/has currently taken in nanoseconds
	// default value of 0 means no duration data is available.
	// RunDuration is not stored on a dataset version, and instead must come from
	// either run state or a cache of run state
	RunDuration int64 `json:"runDuration,omitempty"`
}

// NewVersionInfoFromRef creates a sparse-populated VersionInfo from a dsref.Ref
func NewVersionInfoFromRef(ref Ref) VersionInfo {
	return VersionInfo{
		InitID:    ref.InitID,
		Username:  ref.Username,
		ProfileID: ref.ProfileID,
		Name:      ref.Name,
		Path:      ref.Path,
	}
}

// SimpleRef returns a simple dsref.Ref
func (v VersionInfo) SimpleRef() Ref {
	return Ref{
		InitID:    v.InitID,
		Username:  v.Username,
		ProfileID: v.ProfileID,
		Name:      v.Name,
		Path:      v.Path,
	}
}

// Alias returns the alias components of a Ref as a string
func (v *VersionInfo) Alias() string {
	s := v.Username
	if v.Name != "" {
		s += "/" + v.Name
	}
	return s
}

// ConvertDatasetToVersionInfo assigns values form a dataset to a VersionInfo
// This function is a shim while we work on building up dscache as a store of
// VersionInfo.
//
// Deprecated: Don't use this function for new code. Instead reference a
// VersionInfo that is stored somewhere, or write a function that builds a
// VersionInfo without needing a dataset
func ConvertDatasetToVersionInfo(ds *dataset.Dataset) VersionInfo {
	vi := VersionInfo{
		Username:  ds.Peername,
		ProfileID: ds.ProfileID,
		Name:      ds.Name,
		Path:      ds.Path,
	}
	if ds.Commit != nil {
		vi.CommitTime = ds.Commit.Timestamp
		vi.CommitTitle = ds.Commit.Title
		vi.CommitMessage = ds.Commit.Message
		vi.RunID = ds.Commit.RunID
	}
	if ds.Meta != nil {
		vi.MetaTitle = ds.Meta.Title
		if ds.Meta.Theme != nil {
			vi.ThemeList = strings.Join(ds.Meta.Theme, ",")
		}
	}

	if ds.Structure != nil {
		vi.BodyFormat = ds.Structure.Format
		vi.BodySize = ds.Structure.Length
		vi.BodyRows = ds.Structure.Entries
		vi.NumErrors = ds.Structure.ErrCount
	}

	return vi
}
