package dsref

// Ref is a reference to a dataset
type Ref struct {
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
	return r.Username == "" && r.ProfileID == "" && r.Name == "" && r.Path == ""
}

// Equals returns whether the reference equals another
func (r Ref) Equals(t Ref) bool {
	return r.Username == t.Username && r.ProfileID == t.ProfileID && r.Name == t.Name && r.Path == t.Path
}
