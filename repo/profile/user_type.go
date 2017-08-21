package profile

import (
	"encoding/json"
	"fmt"
)

type UserType int

const (
	UserTypeNone UserType = iota
	UserTypeUser
	UserTypeCommunity
)

func (ut UserType) String() string {
	switch ut {
	case UserTypeNone:
		return "none"
	case UserTypeUser:
		return "user"
	case UserTypeCommunity:
		return "community"
	}

	return "unknown"
}

func (ut UserType) MarshalJSON() ([]byte, error) {
	s, ok := map[UserType]string{UserTypeNone: "none", UserTypeUser: "user", UserTypeCommunity: "community"}[ut]
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

	got, ok := map[string]UserType{"none": UserTypeNone, "user": UserTypeUser, "community": UserTypeCommunity}[s]
	if !ok {
		return fmt.Errorf("invalid UserType %q", s)
	}

	*ut = got
	return nil
}
