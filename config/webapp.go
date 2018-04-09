package config

// DefaultWebappPort is local the port webapp serves on by default
var DefaultWebappPort = "2505"

// Webapp configures the qri webapp service
type Webapp struct {
	Enabled bool
	Port    string
	// token for analytics tracking
	AnalyticsToken string
	// EntrypointHash is a hash of the compiled webapp (the output of running webpack https://github.com/qri-io/frontend)
	// this is fetched and replaced via dnslink when the webapp server starts
	// the value provided here is just a sensible fallback for when dnslink lookup fails,
	// pointing to a known prior version of the the webapp
	EntrypointHash string
}

// Default creates a new default Webapp configuration
func (Webapp) Default() *Webapp {
	return &Webapp{
		Enabled:        true,
		Port:           DefaultWebappPort,
		EntrypointHash: "QmewSfmnridhYdwLc9nGVwFNhMofAjPf4vxMMUYC6QDEjm",
	}
}
