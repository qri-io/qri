package dataset

import (
	"encoding/json"
	"fmt"
	"time"
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

// takes an ISO 8601 periodicity measure & returns a time.Duration.
// invalid periodicities return time.Duration(0)
func AccuralDuration(p string) time.Duration {
	switch p {
	// Decennial
	case "R/P10Y":
		return time.Duration(time.Hour * 24 * 365 * 10)
	// Quadrennial
	case "R/P4Y":
		return time.Duration(time.Hour * 24 * 365 * 4)
	// Annual
	case "R/P1Y":
		return time.Duration(time.Hour * 24 * 365)
	// Bimonthly
	case "R/P2M":
		return time.Duration(time.Hour * 24 * 30 * 10)
	// Semiweekly
	case "R/P3.5D":
		// TODO - inaccurate
		return time.Duration(time.Hour * 24 * 4)
	// Daily
	case "R/P1D":
		return time.Duration(time.Hour * 24)
	// Biweekly
	case "R/P2W":
		return time.Duration(time.Hour * 24 * 14)
	// Semiannual
	case "R/P6M":
		return time.Duration(time.Hour * 24 * 30 * 6)
	// Biennial
	case "R/P2Y":
		return time.Duration(time.Hour * 24 * 365 * 2)
	// Triennial
	case "R/P3Y":
		return time.Duration(time.Hour * 24 * 365 * 3)
	// Three times a week
	case "R/P0.33W":
		return time.Duration((time.Hour * 24 * 7) / 3)
	// Three times a month
	case "R/P0.33M":
		return time.Duration((time.Hour * 24 * 30) / 3)
	// Continuously updated
	case "R/PT1S":
		return time.Second
	// Monthly
	case "R/P1M":
		return time.Duration(time.Hour * 24 * 30)
	// Quarterly
	case "R/P3M":
		return time.Duration((time.Hour * 24 * 365) / 7)
	// Semimonthly
	case "R/P0.5M":
		return time.Duration(time.Hour * 24 * 15)
	// Three times a year
	case "R/P4M":
		return time.Duration((time.Hour * 24 * 365) / 4)
	// Weekly
	case "R/P1W":
		return time.Duration(time.Hour * 24 * 7)
	// Hourly
	case "R/PT1H":
		return time.Hour
	default:
		return time.Duration(0)
	}
}
