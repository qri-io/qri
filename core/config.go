package core

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/config"
)

var (
	// Config is the global configuration object
	Config *config.Config
	// ConfigFilepath is the default location for a config file
	ConfigFilepath string
)

// SaveConfig is a function that updates the configuration file
var SaveConfig = func() error {
	if err := Config.WriteToFile(ConfigFilepath); err != nil {
		return fmt.Errorf("error saving profile: %s", err)
	}
	return nil
}

// LoadConfig loads the global default configuration
func LoadConfig(path string) (err error) {
	var cfg *config.Config
	cfg, err = config.ReadFromFile(path)

	if err == nil && cfg.Profile == nil {
		err = fmt.Errorf("missing profile")
		return
	}

	if err != nil {
		err = fmt.Errorf(`couldn't read config file. error: %s, path: %s`, err.Error(), path)
		return
	}

	// configure logging straight away
	if cfg != nil && cfg.Logging != nil {
		for name, level := range cfg.Logging.Levels {
			golog.SetLogLevel(name, level)
		}
	}

	Config = cfg

	return err
}

// GetConfigParams are the params needed to format/specify the fields in bytes returned from the GetConfig function
type GetConfigParams struct {
	WithPrivateKey bool
	Format         string
	Concise        bool
	Field          string
}

// GetConfig returns the Config, or one of the specified fields of the Config, as a slice of bytes
// the bytes can be formatted as json, concise json, or yaml
func GetConfig(p *GetConfigParams, res *[]byte) error {
	var (
		err    error
		cfg    = &config.Config{}
		encode interface{}
	)

	if !p.WithPrivateKey {
		cfg = Config.WithoutPrivateValues()
	} else {
		cfg = Config.Copy()
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

// SetConfig validates and saves the config (passed in as `res`)
func SetConfig(res *config.Config) error {
	if err := res.Validate(); err != nil {
		return fmt.Errorf("error validating config: %s", err)
	}
	if err := SaveConfig(); err != nil {
		return fmt.Errorf("error saving config: %s", err)
	}

	Config = res.WithPrivateValues(Config)

	return nil
}
