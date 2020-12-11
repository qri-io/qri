package dsref

import (
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/qri-io/qfs/muxfs"
)

// Ref is a reference to a dataset
type Ref struct {
	// InitID is the canonical identifer for a dataset history
	InitID string `json:"initID,omitempty"`
	// Username of dataset owner
	Username string `json:"username,omitempty"`
	// ProfileID of dataset owner
	ProfileID string `json:"profileID,omitempty"`
	// Unique name reference for this dataset
	Name string `json:"name,omitempty"`
	// Content-addressed path for this dataset
	Path string `json:"path,omitempty"`
}

// Alias returns the alias components of a Ref as a string
func (r Ref) Alias() (s string) {
	return r.Human()
}

// Human returns the human-friendly representation of the reference
// example: some_user/my_dataset
func (r Ref) Human() string {
	s := r.Username
	if r.Name != "" {
		s += "/" + r.Name
	}
	return s
}

// String implements the Stringer interface for Ref
func (r Ref) String() (s string) {
	s = r.Alias()
	if r.ProfileID != "" || r.Path != "" {
		s += "@"
	}
	if r.ProfileID != "" {
		s += r.ProfileID
	}
	if r.Path != "" {
		s += r.Path
	}
	return s
}

// IsEmpty returns whether the reference is empty
func (r Ref) IsEmpty() bool {
	return r.InitID == "" && r.Username == "" && r.ProfileID == "" && r.Name == "" && r.Path == ""
}

// Complete returns true if all fields are populated
func (r Ref) Complete() bool {
	return r.InitID != "" && r.Username != "" && r.ProfileID != "" && r.Name != "" && r.Path != ""
}

// Equals returns whether the reference equals another
func (r Ref) Equals(t Ref) bool {
	return r.InitID == t.InitID &&
		r.Username == t.Username &&
		r.ProfileID == t.ProfileID &&
		r.Name == t.Name &&
		r.Path == t.Path
}

// Copy duplicates a reference
func (r Ref) Copy() Ref {
	return Ref{
		InitID:    r.InitID,
		Username:  r.Username,
		ProfileID: r.ProfileID,
		Name:      r.Name,
		Path:      r.Path,
	}
}

// VersionInfo creates a new sparse VersionInfo from a reference
func (r Ref) VersionInfo() VersionInfo {
	return VersionInfo{
		InitID:    r.InitID,
		Username:  r.Username,
		ProfileID: r.ProfileID,
		Name:      r.Name,
		Path:      r.Path,
	}
}

// PathCID gets ref.Path as a CID
func (r Ref) PathCID() (cid.Cid, error) {
	in := strings.TrimLeft(r.Path, "/")
	for _, fsType := range muxfs.KnownFSTypes() {
		in = strings.TrimPrefix(in, fsType)
	}
	in = strings.TrimLeft(in, "/")
	return cid.Parse(in)
}
