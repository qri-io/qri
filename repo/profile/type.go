package profile

import (
	"encoding/json"
	"fmt"
)

// Type enumerates different types of peers
type Type int

const (
	// TypePeer is a single person
	TypePeer Type = iota
	// TypeOrganization represents a group of people
	TypeOrganization
)

// String implements the Stringer interface for Type
func (t Type) String() string {
	switch t {
	case TypePeer:
		return "peer"
	case TypeOrganization:
		return "organization"
	}

	return "unknown"
}

// MarshalJSON implements the json.Marshaler interface for Type
func (t Type) MarshalJSON() ([]byte, error) {
	s, ok := map[Type]string{TypePeer: "peer", TypeOrganization: "organization"}[t]
	if !ok {
		return nil, fmt.Errorf("invalid Type %d", t)
	}

	return []byte(fmt.Sprintf(`"%s"`, s)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for Type
func (t *Type) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("Peer type should be a string, got %s", data)
	}

	got, ok := map[string]Type{"user": TypePeer, "peer": TypePeer, "organization": TypeOrganization}[s]
	if !ok {
		return fmt.Errorf("invalid Type %q", s)
	}

	*t = got
	return nil
}
