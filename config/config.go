// Package config encapsulates qri configuration options & details. configuration is generally stored
// as a .yaml file, or provided at CLI runtime via command a line argument
package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"

	"github.com/qri-io/jsonschema"
	"gopkg.in/yaml.v2"
)

// Config encapsulates all configuration details for qri
type Config struct {
	Profile  *ProfilePod
	Repo     *Repo
	Store    *Store
	P2P      *P2P
	Registry *Registry

	CLI     *CLI
	API     *API
	Webapp  *Webapp
	RPC     *RPC
	Logging *Logging
}

// DefaultConfig gives a new default qri configuration
func DefaultConfig() *Config {
	return &Config{
		Profile:  DefaultProfile(),
		Repo:     DefaultRepo(),
		Store:    DefaultStore(),
		Registry: DefaultRegistry(),

		CLI:     DefaultCLI(),
		API:     DefaultAPI(),
		P2P:     DefaultP2P(),
		Webapp:  DefaultWebapp(),
		RPC:     DefaultRPC(),
		Logging: DefaultLogging(),
	}
}

// SummaryString creates a pretty string summarizing the
// configuration, useful for log output
func (cfg Config) SummaryString() (summary string) {
	summary = "\n"
	if cfg.Profile != nil {
		summary += fmt.Sprintf("peername:\t%s\nprofileID:\t%s\n", cfg.Profile.Peername, cfg.Profile.ID)
	}

	if cfg.API != nil && cfg.API.Enabled {
		summary += fmt.Sprintf("API port:\t%d\n", cfg.API.Port)
	}

	if cfg.RPC != nil && cfg.RPC.Enabled {
		summary += fmt.Sprintf("RPC port:\t%d\n", cfg.RPC.Port)
	}

	if cfg.Webapp != nil && cfg.Webapp.Enabled {
		summary += fmt.Sprintf("Webapp port:\t%d\n", cfg.Webapp.Port)
	}

	return summary
}

// ReadFromFile reads a YAML configuration file from path
func ReadFromFile(path string) (cfg *Config, err error) {
	var data []byte

	data, err = ioutil.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	cfg = &Config{}
	err = yaml.Unmarshal(data, cfg)
	return
}

// WriteToFile encodes a configration to YAML and writes it to path
func (cfg Config) WriteToFile(path string) error {
	// Never serialize the address mapping to the configuration file.
	prev := cfg.Profile.Addresses
	cfg.Profile.Addresses = nil
	defer func() { cfg.Profile.Addresses = prev }()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 06777)
}

// Get a config value with case.insensitive.dot.separated.paths
func (cfg Config) Get(path string) (interface{}, error) {
	v, err := cfg.path(path)
	if err != nil {
		return nil, err
	}
	return v.Interface(), nil
}

// Set a config value with case.insensitive.dot.separated.paths
func (cfg *Config) Set(path string, value interface{}) error {
	v, err := cfg.path(path)
	if err != nil {
		return err
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != v.Kind() {
		return fmt.Errorf("invalid type for config path %s, expected: %s, got: %s", path, v.Kind().String(), rv.Kind().String())
	}

	switch v.Kind() {
	case reflect.Int:
		v.SetInt(rv.Int())
	case reflect.String:
		v.SetString(rv.String())
	case reflect.Bool:
		v.SetBool(rv.Bool())
	}

	return nil
}

func (cfg Config) path(path string) (elem reflect.Value, err error) {
	elem = reflect.ValueOf(cfg)

	for _, sel := range strings.Split(path, ".") {
		sel = strings.ToLower(sel)

		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		switch elem.Kind() {
		case reflect.Struct:
			elem = elem.FieldByNameFunc(func(str string) bool {
				return strings.ToLower(str) == sel
			})
		case reflect.Slice:
			index, err := strconv.Atoi(sel)
			if err != nil {
				return elem, fmt.Errorf("invalid index value: %s", sel)
			}
			elem = elem.Index(index)
		case reflect.Map:
			set := false
			for _, key := range elem.MapKeys() {
				// we only support strings as values
				if strings.ToLower(key.String()) == sel {
					elem = elem.MapIndex(key)
					set = true
					break
				}
			}
			if !set {
				return elem, fmt.Errorf("invalid config path: %s", path)
			}
		}

		if elem.Kind() == reflect.Invalid {
			return elem, fmt.Errorf("invalid config path: %s", path)
		}
	}

	return elem, nil
}

// ImmutablePaths returns a map of paths that should never be modified
func ImmutablePaths() map[string]bool {
	return map[string]bool{
		"p2p.peerid":       true,
		"p2p.pubkey":       true,
		"p2p.privkey":      true,
		"profile.id":       true,
		"profile.privkey":  true,
		"profile.peername": true,
		"profile.created":  true,
		"profile.updated":  true,
	}
}

// valiate is a helper function that wraps json.Marshal an ValidateBytes
// it is used by each struct that is in a Config field (eg API, Profile, etc)
func validate(rs *jsonschema.RootSchema, s interface{}) error {
	strct, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("error marshaling profile to json: %s", err)
	}
	if errors, err := rs.ValidateBytes(strct); len(errors) > 0 {
		return fmt.Errorf("%s", errors[0])
	} else if err != nil {
		return err
	}
	return nil
}

// Validate validates each section of the config struct,
// returning the first error
func (cfg Config) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "config",
    "description": "qri configuration",
    "type": "object",
    "required": ["Profile", "Repo", "Store", "P2P", "CLI", "API", "Webapp", "RPC"],
    "properties" : {
    	"Profile" : { "type":"object" },
    	"Repo" : { "type":"object" },
    	"Store" : { "type":"object" },
    	"P2P" : { "type":"object" },
    	"CLI" : { "type":"object" },
    	"API" : { "type":"object" },
    	"Webapp" : { "type":"object" },
    	"RPC" : { "type":"object" }
    }
  }`)
	if err := validate(schema, &cfg); err != nil {
		return err
	}

	if err := cfg.Profile.Validate(); err != nil {
		return err
	}
	if err := cfg.Repo.Validate(); err != nil {
		return err
	}
	if err := cfg.Store.Validate(); err != nil {
		return err
	}
	if err := cfg.P2P.Validate(); err != nil {
		return err
	}
	if err := cfg.CLI.Validate(); err != nil {
		return err
	}
	if err := cfg.API.Validate(); err != nil {
		return err
	}
	if err := cfg.Webapp.Validate(); err != nil {
		return err
	}
	if err := cfg.RPC.Validate(); err != nil {
		return err
	}
	return cfg.Logging.Validate()
}

// Copy returns a deep copy of the Config struct
func (cfg *Config) Copy() *Config {
	res := &Config{}

	if cfg.Profile != nil {
		res.Profile = cfg.Profile.Copy()
	}
	if cfg.Repo != nil {
		res.Repo = cfg.Repo.Copy()
	}
	if cfg.Store != nil {
		res.Store = cfg.Store.Copy()
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
	if cfg.Webapp != nil {
		res.Webapp = cfg.Webapp.Copy()
	}
	if cfg.RPC != nil {
		res.RPC = cfg.RPC.Copy()
	}
	if cfg.Logging != nil {
		res.Logging = cfg.Logging.Copy()
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
