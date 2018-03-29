package config

// DefaultWebappPort is local the port webapp serves on by default
var DefaultWebappPort = "2505"

// Webapp configures the qri webapp service
type Webapp struct {
	Enabled bool
	Port    string
	// token for analytics tracking
	AnalyticsToken string
	// Scripts is a list of script tags to include the webapp page, useful for using alternative / beta
	// versions of the qri frontend
	Scripts []string
}

// Default creates a new default Webapp configuration
func (Webapp) Default() *Webapp {
	return &Webapp{
		Enabled: true,
		Port:    DefaultWebappPort,
		Scripts: []string{
			// this is fetched and replaced via dnslink when the webapp server starts
			// the value provided here is just a sensible fallback for when dnslink lookup fails,
			// pointing to a known prior version of the the webapp
			"http://localhost:2503/ipfs/QmYDkLrvzDpzDzKeLD3okiQUzSL1ksNsfYU6ZwRYYn8ViS",
		},
	}
}
