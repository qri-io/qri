package api

import (
	"fmt"

	"github.com/qri-io/qri/logging"
)

// server modes
const (
	DEVELOP_MODE    = "develop"
	PRODUCTION_MODE = "production"
	TEST_MODE       = "test"
)

func DefaultConfig() *Config {
	return &Config{
		Logger:      logging.DefaultLogger,
		Mode:        "develop",
		Port:        "8080",
		QriRepoPath: "~/qri",
		FsStorePath: "~/.ipfs",
		Online:      true,
	}
}

// config holds all configuration for the server. It pulls from three places (in order):
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
	Logger logging.Logger
	// operation mode
	Mode string
	// port to listen on, will be read from PORT env variable if present.
	Port string
	// root url for service
	UrlRoot string
	// path to ipfs filestore
	FsStorePath string
	// path to qri repository
	QriRepoPath string
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
	ProxyForceHttps bool
	// token for analytics tracking
	AnalyticsToken string
	// set to true to run entire server with in-memory structures
	MemOnly bool
	// disable networking
	Online bool
	// list of addresses to bootsrap qri peers on
	BoostrapAddrs []string
}

// Validate returns nil if this configuration is valid,
// a descriptive error otherwise
func (cfg *Config) Validate() (err error) {
	// make sure port is set
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	err = requireConfigStrings(map[string]string{
		"PORT": cfg.Port,
	})

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
