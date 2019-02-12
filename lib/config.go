package lib

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/qri-io/qri/config"
)

// GetConfigParams are the params needed to format/specify the fields in bytes returned from the GetConfig function
type GetConfigParams struct {
	WithPrivateKey bool
	Format         string
	Concise        bool
	Field          string
}

// Config provides read & write methods for configuration details
type Config struct {
	cfg      *config.Config
	filePath string
}

// Get returns the Config, or one of the specified fields of the Config, as a slice of bytes
// the bytes can be formatted as json, concise json, or yaml
func (c *Config) Get(p *GetConfigParams, res *[]byte) error {
	var (
		err    error
		cfg    = &config.Config{}
		encode interface{}
	)

	if !p.WithPrivateKey {
		cfg = c.cfg.WithoutPrivateValues()
	} else {
		cfg = c.cfg.Copy()
	}

	encode = cfg

	if p.Field != "" {
		encode, err = cfg.Get(p.Field)
		if err != nil {
			return fmt.Errorf("error getting %s from config: %s", p.Field, err)
		}
	}

	switch p.Format {
	case "json":
		if p.Concise {
			*res, err = json.Marshal(encode)
		} else {
			*res, err = json.MarshalIndent(encode, "", " ")
		}
	case "yaml":
		*res, err = yaml.Marshal(encode)
	}
	if err != nil {
		return fmt.Errorf("error getting config: %s", err)
	}

	return nil
}

// Set validates, updates and saves the config
func (c *Config) Set(update *config.Config) error {
	if err := update.Validate(); err != nil {
		return fmt.Errorf("error validating config: %s", err)
	}

	cfg := update.WithPrivateValues(c.cfg)
	if err := cfg.WriteToFile(c.filePath); err != nil {
		return err
	}
	c.cfg = cfg

	return nil
}
