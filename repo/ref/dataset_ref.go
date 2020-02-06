package reporef

import (
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo/profile"
)

// DatasetRef encapsulates a reference to a dataset. This needs to exist to bind
// ways of referring to a dataset to a dataset itself, as datasets can't easily
// contain their own hash information, and names are unique on a per-repository
// basis.
//
// Deprecated: DatasetRef will be removed in future versions of Qri, use
// dsref.Ref and this package's Info struct instead
type DatasetRef struct {
	// Peername of dataset owner
	Peername string `json:"peername,omitempty"`
	// ProfileID of dataset owner
	ProfileID profile.ID `json:"profileID,omitempty"`
	// Unique name reference for this dataset
	Name string `json:"name,omitempty"`
	// Content-addressed path for this dataset
	Path string `json:"path,omitempty"`
	// FSIPath is this dataset's link to the local filesystem if one exists
	FSIPath string `json:"fsiPath,omitempty"`
	// Dataset is a pointer to the dataset being referenced
	Dataset *dataset.Dataset `json:"dataset,omitempty"`
	// Published indicates whether this reference is listed as an available dataset
	Published bool `json:"published"`
	// If true, this reference doesn't exist locally. Only makes sense if path is set, as this
	// flag refers to specific versions, not to entire dataset histories.
	Foreign bool `json:"foreign,omitempty"`
}

// String implements the Stringer interface for DatasetRef
func (r DatasetRef) String() (s string) {
	s = r.AliasString()
	if r.Path != "" {
		s += "@" + r.Path
	}
	return
}

// DebugString returns a string that describes every field in the reference, useful for tests
func (r DatasetRef) DebugString() string {
	builder := strings.Builder{}
	builder.WriteString("{peername:")
	builder.WriteString(r.Peername)
	builder.WriteString(",profileID:")
	builder.WriteString(r.ProfileID.String())
	builder.WriteString(",name:")
	builder.WriteString(r.Name)
	if r.Path != "" {
		builder.WriteString(",path:")
		builder.WriteString(r.Path)
	}
	if r.FSIPath != "" {
		builder.WriteString(",fsiPath:")
		builder.WriteString(r.FSIPath)
	}
	if r.Published {
		builder.WriteString(",published:true")
	}
	if r.Foreign {
		builder.WriteString(",foreign:true")
	}
	builder.WriteString("}")
	return builder.String()
}

// Absolute implements the same thing as String(), but append ProfileID if it exist
func (r DatasetRef) Absolute() (s string) {
	s = r.AliasString()
	if r.ProfileID.String() != "" || r.Path != "" {
		s += "@"
	}
	if r.ProfileID.String() != "" {
		s += r.ProfileID.String()
	}
	if r.Path != "" {
		s += r.Path
	}
	return
}

// AliasString returns the alias components of a DatasetRef as a string
func (r DatasetRef) AliasString() (s string) {
	s = r.Peername
	if r.Name != "" {
		s += "/" + r.Name
	}
	return
}

// Complete returns true if a dataset has Peername, Name, ProfileID and Path
// properties set
func (r DatasetRef) Complete() bool {
	return r.Peername != "" && r.ProfileID != "" && r.Name != "" && r.Path != ""
}

// Match checks returns true if Peername and Name are equal,
// and/or path is equal
func (r DatasetRef) Match(b DatasetRef) bool {
	// fmt.Printf("\nr.Peername: %s b.Peername: %s\n", r.Peername, b.Peername)
	// fmt.Printf("\nr.Name: %s b.Name: %s\n", r.Name, b.Name)
	return (r.Path != "" && b.Path != "" && r.Path == b.Path) || (r.ProfileID == b.ProfileID || r.Peername == b.Peername) && r.Name == b.Name
}

// Equal returns true only if Peername Name and Path are equal
func (r DatasetRef) Equal(b DatasetRef) bool {
	return r.Peername == b.Peername && r.ProfileID == b.ProfileID && r.Name == b.Name && r.Path == b.Path
}

// IsPeerRef returns true if only Peername is set
func (r DatasetRef) IsPeerRef() bool {
	return (r.Peername != "" || r.ProfileID != "") && r.Name == "" && r.Path == "" && r.Dataset == nil
}

// IsEmpty returns true if none of it's fields are set
func (r DatasetRef) IsEmpty() bool {
	return r.Equal(DatasetRef{})
}
