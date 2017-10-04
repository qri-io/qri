package profile

import (
	"encoding/json"
	"fmt"
)

type UserType int

const (
	UserTypeUser UserType = iota
	UserTypeOrganization
)

func (ut UserType) String() string {
	switch ut {
	case UserTypeUser:
		return "user"
	case UserTypeOrganization:
		return "organization"
	}

	return "unknown"
}

func (ut UserType) MarshalJSON() ([]byte, error) {
	s, ok := map[UserType]string{UserTypeNone: "none", UserTypeUser: "user", UserTypeOrganization: "organization"}[ut]
	if !ok {
		return nil, fmt.Errorf("invalid UserType %d", ut)
	}

	return []byte(fmt.Sprintf(`"%s"`, s)), nil
}

func (ut *UserType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("User type should be a string, got %s", data)
	}

	got, ok := map[string]UserType{"none": UserTypeNone, "user": UserTypeUser, "organization": UserTypeOrganization}[s]
	if !ok {
		return fmt.Errorf("invalid UserType %q", s)
	}

	*ut = got
	return nil
}
