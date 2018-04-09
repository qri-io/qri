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
	Profile *Profile
	Repo    *Repo
	Store   *Store
	P2P     *P2P

	CLI     *CLI
	API     *API
	Webapp  *Webapp
	RPC     *RPC
	Logging *Logging
}

// DefaultConfig gives a new default qri configuration
func DefaultConfig() *Config {
	return &Config{
		Profile: DefaultProfile(),
		Repo:    DefaultRepo(),
		Store:   DefaultStore(),

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
		summary += fmt.Sprintf("API port:\t%s\n", cfg.API.Port)
	}

	if cfg.RPC != nil && cfg.RPC.Enabled {
		summary += fmt.Sprintf("RPC port:\t%s\n", cfg.RPC.Port)
	}

	if cfg.Webapp != nil && cfg.Webapp.Enabled {
		summary += fmt.Sprintf("Webapp port:\t%s\n", cfg.Webapp.Port)
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
