package config

// DefaultAPIPort is local the port webapp serves on by default
var DefaultAPIPort = "2503"

// API holds configuration for the qri JSON api
type API struct {
	Enabled bool
	// APIPort specifies the port to listen for JSON API calls
	Port string
	// read-only mode
	ReadOnly bool
	// URLRoot is the base url for this server
	URLRoot string
	// TLS enables https via letsEyncrypt
	TLS bool
	// if true, requests that have X-Forwarded-Proto: http will be redirected
	// to their https variant
	ProxyForceHTTPS bool
	// support CORS signing from a list of origins
	AllowedOrigins []string
}

// Default returns the default configuration details
func (API) Default() *API {
	return &API{
		Enabled: true,
		Port:    DefaultAPIPort,
		TLS:     true,
		AllowedOrigins: []string{
			"http://localhost:2505",
		},
	}
}
