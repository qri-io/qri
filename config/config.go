// Package config encapsulates qri configuration options & details. configuration is generally stored
// as a .yaml file, or provided at CLI runtime via command a line argument
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/fill"
)

// CurrentConfigRevision is the latest configuration revision configurations
// that don't match this revision number should be migrated up
const CurrentConfigRevision = 3

// Config encapsulates all configuration details for qri
type Config struct {
	path string

	Revision    int
	Profile     *ProfilePod
	Repo        *Repo
	Filesystems []qfs.Config
	P2P         *P2P
	Stats       *Stats

	Registry *Registry
	Remotes  *Remotes
	Remote   *Remote

	CLI     *CLI
	API     *API
	RPC     *RPC
	Logging *Logging
}

// SetArbitrary is an interface implementation of base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct.
// This simply ignores all additional fields at read time.
func (cfg *Config) SetArbitrary(key string, val interface{}) error {
	return nil
}

// NOTE: The configuration returned by DefaultConfig is insufficient, as is, to run a functional
// qri node. In particular, it lacks cryptographic keys and a peerID, which are necessary to
// join the p2p network. However, these are very expensive to create, so they shouldn't be added
// to the DefaultConfig, which only does the bare minimum necessary to construct the object. In
// real use, the only places a Config object comes from are the cmd/setup command, which builds
// upon DefaultConfig by adding p2p data, and LoadConfig, which parses a serialized config file
// from the user's repo.

// DefaultConfig gives a new configuration with simple, default settings
func DefaultConfig() *Config {
	return &Config{
		Revision:    CurrentConfigRevision,
		Profile:     DefaultProfile(),
		Repo:        DefaultRepo(),
		Filesystems: DefaultFilesystems(),
		P2P:         DefaultP2P(),
		Stats:       DefaultStats(),

		Registry: DefaultRegistry(),
		// default to no configured remotes

		CLI:     DefaultCLI(),
		API:     DefaultAPI(),
		RPC:     DefaultRPC(),
		Logging: DefaultLogging(),
	}
}

// SummaryString creates a pretty string summarizing the
// configuration, useful for log output
// TODO (b5): this summary string doesn't confirm these services are actually
// running. we should move this elsewhere
func (cfg Config) SummaryString() (summary string) {
	summary = "\n"
	if cfg.Profile != nil {
		summary += fmt.Sprintf("peername:\t%s\nprofileID:\t%s\n", cfg.Profile.Peername, cfg.Profile.ID)
	}

	if cfg.API != nil && cfg.API.Enabled {
		summary += fmt.Sprintf("API address:\t%s\n", cfg.API.Address)
		if cfg.API.WebsocketAddress != "" {
			summary += fmt.Sprintf("Websocket address:\t%s\n", cfg.API.WebsocketAddress)
		}
	}

	if cfg.RPC != nil && cfg.RPC.Enabled {
		summary += fmt.Sprintf("RPC address:\t%s\n", cfg.RPC.Address)
	}

	return summary
}

// ReadFromFile reads a YAML configuration file from path
func ReadFromFile(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fields := make(map[string]interface{})
	if err = yaml.Unmarshal(data, &fields); err != nil {
		return nil, err
	}

	cfg := &Config{path: path}

	if rev, ok := fields["revision"]; ok {
		cfg.Revision = (int)(rev.(float64))
	}
	if err = fill.Struct(fields, cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// SetPath assigns unexported filepath to write config to
func (cfg *Config) SetPath(path string) {
	cfg.path = path
}

// Path gives the unexported filepath for a config
func (cfg Config) Path() string {
	return cfg.path
}

// WriteToFile encodes a configration to YAML and writes it to path
func (cfg Config) WriteToFile(path string) error {
	// Never serialize the address mapping to the configuration file.
	prev := cfg.Profile.PeerIDs
	cfg.Profile.NetworkAddrs = nil
	cfg.Profile.Online = false
	cfg.Profile.PeerIDs = nil
	defer func() { cfg.Profile.PeerIDs = prev }()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 06777)
}

// Get a config value with case.insensitive.dot.separated.paths
func (cfg Config) Get(path string) (interface{}, error) {
	return fill.GetPathValue(path, cfg)
}

// Set a config value with case.insensitive.dot.separated.paths
func (cfg *Config) Set(path string, value interface{}) error {
	return fill.SetPathValue(path, value, cfg)
}

// ImmutablePaths returns a map of paths that should never be modified
func ImmutablePaths() map[string]bool {
	return map[string]bool{
		"p2p.peerid":      true,
		"p2p.privkey":     true,
		"profile.id":      true,
		"profile.privkey": true,
		"profile.created": true,
		"profile.updated": true,
	}
}

// valiate is a helper function that wraps json.Marshal an ValidateBytes
// it is used by each struct that is in a Config field (eg API, Profile, etc)
func validate(rs *jsonschema.Schema, s interface{}) error {
	ctx := context.Background()
	strct, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("error marshaling profile to json: %s", err)
	}
	if errors, err := rs.ValidateBytes(ctx, strct); len(errors) > 0 {
		return fmt.Errorf("%s", errors[0])
	} else if err != nil {
		return err
	}
	return nil
}

type validator interface {
	Validate() error
}

// Validate validates each section of the config struct,
// returning the first error
func (cfg Config) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "config",
    "description": "qri configuration",
    "type": "object",
    "required": ["Profile", "Repo", "Filesystems", "P2P", "CLI", "API", "RPC"],
    "properties" : {
			"Profile" : { "type":"object" },
			"Repo" : { "type":"object" },
			"Filesystems" : { "type":"array" },
			"P2P" : { "type":"object" },
			"CLI" : { "type":"object" },
			"API" : { "type":"object" },
			"RPC" : { "type":"object" }
    }
  }`)
	if err := validate(schema, &cfg); err != nil {
		return fmt.Errorf("config validation error: %s", err)
	}

	validators := []validator{
		cfg.Profile,
		cfg.Repo,
		cfg.P2P,
		cfg.CLI,
		cfg.API,
		cfg.RPC,
		cfg.Logging,
	}
	for _, val := range validators {
		// we need to check here because we're potentially calling methods on nil
		// values that don't handle a nil receiver gracefully.
		// https://tour.golang.org/methods/12
		// https://groups.google.com/forum/#!topic/golang-nuts/wnH302gBa4I/discussion
		// TODO (b5) - make validate methods handle being nil
		if !reflect.ValueOf(val).IsNil() {
			if err := val.Validate(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Copy returns a deep copy of the Config struct
func (cfg *Config) Copy() *Config {
	res := &Config{
		Revision: cfg.Revision,
	}
	if cfg.path != "" {
		res.path = cfg.path
	}
	if cfg.Profile != nil {
		res.Profile = cfg.Profile.Copy()
	}
	if cfg.Repo != nil {
		res.Repo = cfg.Repo.Copy()
	}
	if cfg.P2P != nil {
		res.P2P = cfg.P2P.Copy()
	}
	if cfg.Registry != nil {
		res.Registry = cfg.Registry.Copy()
	}
	if cfg.CLI != nil {
		res.CLI = cfg.CLI.Copy()
	}
	if cfg.API != nil {
		res.API = cfg.API.Copy()
	}
	if cfg.RPC != nil {
		res.RPC = cfg.RPC.Copy()
	}
	if cfg.Remote != nil {
		res.Remote = cfg.Remote.Copy()
	}
	if cfg.Remotes != nil {
		res.Remotes = cfg.Remotes.Copy()
	}
	if cfg.Logging != nil {
		res.Logging = cfg.Logging.Copy()
	}
	if cfg.Stats != nil {
		res.Stats = cfg.Stats.Copy()
	}
	if cfg.Filesystems != nil {
		for _, fs := range cfg.Filesystems {
			res.Filesystems = append(res.Filesystems, fs)
		}
	}

	return res
}

// WithoutPrivateValues returns a deep copy of the receiver with the private values removed
func (cfg *Config) WithoutPrivateValues() *Config {
	res := cfg.Copy()

	res.Profile.PrivKey = ""
	res.P2P.PrivKey = ""

	return res
}

// WithPrivateValues returns a deep copy of the receiver with the private values from
// the *Config passed in from the params
func (cfg *Config) WithPrivateValues(p *Config) *Config {
	res := cfg.Copy()

	res.Profile.PrivKey = p.Profile.PrivKey
	res.P2P.PrivKey = p.P2P.PrivKey

	return res
}

// DefaultFilesystems is the default filesystem stack
func DefaultFilesystems() []qfs.Config {
	return []qfs.Config{
		{
			Type: "ipfs",
			Config: map[string]interface{}{
				"path": filepath.Join(".", "ipfs"),
			},
		},
		{Type: "local"},
		{Type: "http"},
	}
}
