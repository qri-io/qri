package profile

import (
	"encoding/json"
	"fmt"
)

// UserType enumerates different types of users
type UserType int

const (
	// UserTypeUser is a single person
	UserTypeUser UserType = iota
	// UserTypeOrganization represents a group of people
	UserTypeOrganization
)

// String implements the Stringer interface for UserType
func (ut UserType) String() string {
	switch ut {
	case UserTypeUser:
		return "user"
	case UserTypeOrganization:
		return "organization"
	}

	return "unknown"
}

// MarshalJSON implements the json.Marshaler interface for UserType
func (ut UserType) MarshalJSON() ([]byte, error) {
	s, ok := map[UserType]string{UserTypeUser: "user", UserTypeOrganization: "organization"}[ut]
	if !ok {
		return nil, fmt.Errorf("invalid UserType %d", ut)
	}

	return []byte(fmt.Sprintf(`"%s"`, s)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for UserType
func (ut *UserType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("User type should be a string, got %s", data)
	}

	got, ok := map[string]UserType{"user": UserTypeUser, "organization": UserTypeOrganization}[s]
	if !ok {
		return fmt.Errorf("invalid UserType %q", s)
	}

	*ut = got
	return nil
}
