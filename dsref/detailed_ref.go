package dsref

import (
	"time"
)

// DetailedRef describes a dataset reference and detailed information about it
type DetailedRef struct {
	// Username of dataset owner
	Username string `json:"username,omitempty"`
	// ProfileID of dataset owner
	ProfileID string `json:"profileID,omitempty"`
	// Unique name reference for this dataset
	Name string `json:"name,omitempty"`
	// Content-addressed path for this dataset
	Path string `json:"path,omitempty"`
	// If true, this reference doesn't exist locally
	Foreign bool `json:"foreign,omitempty"`
	// Title from the meta structure
	MetaTitle string `json:"metaTitle,omitempty"`
	// List of themes from the meta structure, comma-separated list
	ThemeList string `json:"themeList,omitempty"`
	// Size of the body in bytes
	BodySize int `json:"bodySize,omitempty"`
	// Num of rows in the body
	BodyRows int `json:"bodyRows,omitempty"`
	// Timestamp field from the commit structure
	CommitTime time.Time `json:"commitTime,omitempty"`
	// Number of errors from the structure
	NumErrors int `json:"numErrors,omitempty"`
	// Number of versions that the dataset has
	NumVersions int `json:"numVersions,omitempty"`
	// FSIPath is this dataset's link to the local filesystem if one exists
	FSIPath string `json:"fsiPath,omitempty"`
}

// SimpleRef returns a simple dsref.Ref
func (r *DetailedRef) SimpleRef() Ref {
	return Ref{
		Username:  r.Username,
		ProfileID: r.ProfileID,
		Name:      r.Name,
		Path:      r.Path,
	}
}

// Alias returns the alias components of a Ref as a string
func (r *DetailedRef) Alias() string {
	s := r.Username
	if r.Name != "" {
		s += "/" + r.Name
	}
	return s
}
