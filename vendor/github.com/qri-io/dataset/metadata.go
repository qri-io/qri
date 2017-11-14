package dataset

import (
	"encoding/json"
	"fmt"
)

// Current version of the specification
const version = "0.0.1"

// VersionNumber is a semantic major.minor.patch
// TODO - make Version enforce this format
type VersionNumber string

// User is a placholder for talking about people, groups, organizations
type User struct {
	Id       string `json:"id,omitempty"`
	Fullname string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
}

// License represents a legal licensing agreement
type License struct {
	Type string `json:"type"`
	Url  string `json:"url"`
}

// private struct for marshaling
type _license License

// MarshalJSON satisfies the json.Marshaller interface
func (l License) MarshalJSON() ([]byte, error) {
	if l.Type != "" && l.Url == "" {
		return []byte(fmt.Sprintf(`"%s"`, l.Type)), nil
	}

	return json.Marshal(_license(l))
}

// UnmarshalJSON satisfies the json.Unmarshaller interface
func (l *License) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*l = License{Type: s}
		return nil
	}

	_l := &_license{}
	if err := json.Unmarshal(data, _l); err != nil {
		return fmt.Errorf("error parsing license from json: %s", err.Error())
	}
	*l = License(*_l)

	return nil
}

// Citation is a place that this dataset drew it's information from
type Citation struct {
	Name  string `json:"name,omitempty"`
	Url   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// Theme is pulled from the Project Open Data Schema version 1.1
type Theme struct {
	Description     string `json:"description,omitempty"`
	DisplayName     string `json:"display_name,omitempty"`
	ImageDisplayUrl string `json:"image_display_url,omitempty"`
	Id              string `json:"id,omitempty"`
	Name            string `json:"name,omitempty"`
	Title           string `json:"title,omitempty"`
}
