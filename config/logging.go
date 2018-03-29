package config

// Logging encapsulates logging configuration
type Logging struct {
	// Levels is a map of package_name : log_level (one of [info, error, debug, warn])
	Levels map[string]string
}

// Default produces a new default logging configuration
func (Logging) Default() *Logging {
	return &Logging{
		Levels: map[string]string{
			"qriapi": "info",
		},
	}
}
