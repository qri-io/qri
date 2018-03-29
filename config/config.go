// Package config encapsulates qri configuration options & details. configuration is generally stored
// as a .yaml file, or provided at CLI runtime via command a line argument
package config

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"

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

// Default gives a new default qri configuration
func (Config) Default() *Config {
	return &Config{
		Profile: Profile{}.Default(),
		Repo:    Repo{}.Default(),
		Store:   Store{}.Default(),

		CLI:     CLI{}.Default(),
		API:     API{}.Default(),
		P2P:     P2P{}.Default(),
		Webapp:  Webapp{}.Default(),
		RPC:     RPC{}.Default(),
		Logging: Logging{}.Default(),
	}
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

// // Config configures the behavior of qri
// // TODO - move all Config related stuff that isn't a command into a different package.
// type Config struct {
// 	// Initialized is a flag for when this repo has been properly initialized at least once.
// 	// used to check weather default datasets should be added or not
// 	Initialized bool
// 	// Identity Configuration details
// 	// Identity IdentityCfg
// 	// List of nodes to boostrap to
// 	Bootstrap []string
// 	// PeerID lists this current peer ID
// 	PeerID string
// 	// PrivateKey is a base-64 encoded private key
// 	PrivateKey string
// 	// IPFSPath is the local path to an IPFS directory
// 	IPFSPath string
// 	// Datastore configuration details
// 	// Datastore       DatastoreCfg
// 	// DefaultDatasets is a list of dataset references to grab on initially joining the network
// 	DefaultDatasets []string
// }

// func defaultCfgBytes() []byte {
// 	cfg := config.Config{}.Default()
// 	data, err := yaml.Marshal(cfg)
// 	ExitIfErr(err)
// 	return data
// }
