package dsref

import (
	"time"
)

// VersionInfo is the way that everything talks about information about datasets in the cases
// that an entire Dataset is not being returned. Used as an output value for many methods, such
// as List, and Log.
// If any fields are added to this struct, keep it in sync with:
//   dscache/def.fbs       dscache
//   dscache/builder.go    func AddDsVersionInfo
//   dscache/convert.go    dsInfo struct
//   dscache/fill_info.go  func fillInfoForDatasets
//   repo/ref/convert.go   func ConvertFromDatasetRef

// VersionInfo describes a dataset reference and detailed information about it
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
	BodyFormat string `json:"bodyFromat,omitempty"`
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
}

// SimpleRef returns a simple dsref.Ref
func (v *VersionInfo) SimpleRef() Ref {
	return Ref{
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
