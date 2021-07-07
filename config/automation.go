package config

import (
	"fmt"

	"github.com/dustin/go-humanize"
)

// Automation encapsulates configuration for the automation subsystem
type Automation struct {
	Enabled         bool
	RunStoreMaxSize string
}

// DefaultAutomation constructs an automation configuration with standard values
func DefaultAutomation() *Automation {
	return &Automation{
		Enabled:         true,
		RunStoreMaxSize: "100Mb",
	}
}

// SetArbitrary is an interface implementation of base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct.
// This simply ignores all additional fields at read time.
func (a *Automation) SetArbitrary(key string, val interface{}) error {
	return nil
}

// Validate ensures a correct Automation configuration
func (a *Automation) Validate() error {
	if a.RunStoreMaxSize != "unlimited" && a.RunStoreMaxSize != "" {
		if _, err := humanize.ParseBytes(a.RunStoreMaxSize); err != nil {
			return fmt.Errorf("invalid RunStoreMaxSize: %w", err)
		}
	} else if a.RunStoreMaxSize != "unlimited" {
		return fmt.Errorf("invalid RunStoreMaxSize value: %s", a.RunStoreMaxSize)
	}

	return nil
}

// Copy creates a shallow copy of Automation
func (a *Automation) Copy() *Automation {
	return &Automation{
		Enabled:         a.Enabled,
		RunStoreMaxSize: a.RunStoreMaxSize,
	}
}
