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
	s = r.Username
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
