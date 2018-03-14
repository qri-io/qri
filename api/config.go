package api

import (
	"fmt"

	"github.com/qri-io/qri/p2p"
)

// server modes
const (
	ModeDevelop       = "develop"
	ModeProduction    = "production"
	ModeTest          = "test"
	DefaultAPIPort    = "2503"
	DefaultRPCPort    = "2504"
	DefaultWebappPort = "2505"
)

// DefaultConfig returns the default configuration details
func DefaultConfig() *Config {
	return &Config{
		Mode:           "develop",
		APIPort:        DefaultAPIPort,
		RPCPort:        DefaultRPCPort,
		WebappPort:     DefaultWebappPort,
		AllowedOrigins: []string{"http://localhost:2505"},
		Online:         true,
		WebappScripts: []string{
			// this is fetched and replaced via dnslink when the webapp server starts
			// the value provided here is just a sensible fallback for when dnslink lookup fails,
			// pointing to a known prior version of the the webapp
			"http://localhost:2503/ipfs/QmYDkLrvzDpzDzKeLD3okiQUzSL1ksNsfYU6ZwRYYn8ViS",
		},
	}
}

// Config holds all configuration for the server. It pulls from three places (in order):
// 		1. environment variables
// 		2. .[MODE].env OR .env
//
// globally-set env variables win.
// it's totally fine to not have, say, .env.develop defined, and just
// rely on a base ".env" file. But if you're in production mode & ".env.production"
// exists, that will be read *instead* of .env
//
// configuration is read at startup and cannot be alterd without restarting the server.
type Config struct {
	// operation mode
	Mode string
	// URLRoot is the base url for this server
	URLRoot string
	// DNS service discovery. Should be either "env" or "dns", default is env
	GetHostsFrom string
	// Public Key to use for signing metablocks. required.
	PublicKey string
	// TLS (HTTPS) enable support via LetsEncrypt, default false
	// should be true in production
	TLS bool
	// support CORS signing from a list of origins
	AllowedOrigins []string
	// if true, requests that have X-Forwarded-Proto: http will be redirected
	// to their https variant
	ProxyForceHTTPS bool
	// token for analytics tracking
	AnalyticsToken string
	// set to true to run entire server with in-memory structures
	MemOnly bool
	// disable networking
	Online bool
	// APIPort specifies the port to listen for JSON API calls
	APIPort string
	// RPCPort specifies the port to listen for RPC calls on, if empty server will not register a RPC listener
	RPCPort string
	// set to to disable webapp
	WebappPort string
	// list of addresses to bootsrap qri peers on
	BoostrapAddrs []string
	// WebappScripts is a list of script tags to include the webapp page, useful for using alternative / beta
	// versions of the qri frontend
	WebappScripts []string
	// PostP2POnlineHook is a chance to call a function after starting P2P services
	PostP2POnlineHook func(*p2p.QriNode)
}

// Validate returns nil if this configuration is valid,
// a descriptive error otherwise
// TODO - this func has been reduced to a noop, let's make it work again
func (cfg *Config) Validate() (err error) {
	// make sure port is set
	// if cfg.APIPort == "" {
	// 	cfg.APIPort = "2503"
	// }

	// err = requireConfigStrings(map[string]string{
	// 	"PORT": cfg.APIPort,
	// })

	return
}

// requireConfigStrings panics if any of the passed in values aren't set
func requireConfigStrings(values map[string]string) error {
	for key, value := range values {
		if value == "" {
			return fmt.Errorf("%s env variable or config key must be set", key)
		}
	}

	return nil
}
