package lib

import (
	"encoding/json"
	"fmt"
	"net/rpc"

	"github.com/ghodss/yaml"
	"github.com/qri-io/qri/config"
)

// var (
// 	// Config is the global configuration object
// 	Config *config.Config
// 	// ConfigFilepath is the default location for a config file
// 	ConfigFilepath string
// )

// // SaveConfig is a function that updates the configuration file
// var SaveConfig = func() error {
// 	if err := Config.WriteToFile(ConfigFilepath); err != nil {
// 		return fmt.Errorf("error saving profile: %s", err)
// 	}
// 	return nil
// }

// // LoadConfig loads the global default configuration
// func LoadConfig(streams ioes.IOStreams, path string) (err error) {
// 	cfg, err := config.ReadFromFile(path)
// 	if err != nil {
// 		return err
// 	}

// 	if cfg.Profile == nil {
// 		err = fmt.Errorf("missing profile")
// 		return err
// 	}

// 	// configure logging straight away
// 	if cfg.Logging != nil {
// 		for name, level := range cfg.Logging.Levels {
// 			golog.SetLogLevel(name, level)
// 		}
// 	}

// 	Config = cfg

// 	migrated, err := migrate.RunMigrations(streams, cfg)
// 	if err != nil {
// 		return err
// 	}

// 	if migrated {
// 		return SaveConfig()
// 	}

// 	return nil
// }

// // GetConfigParams are the params needed to format/specify the fields in bytes returned from the GetConfig function
// type GetConfigParams struct {
// 	WithPrivateKey bool
// 	Format         string
// 	Concise        bool
// 	Field          string
// }

// // GetConfig returns the Config, or one of the specified fields of the Config, as a slice of bytes
// // the bytes can be formatted as json, concise json, or yaml
// func GetConfig(p *GetConfigParams, res *[]byte) error {
// 	var (
// 		err    error
// 		cfg    = &config.Config{}
// 		encode interface{}
// 	)

// 	if !p.WithPrivateKey {
// 		cfg = Config.WithoutPrivateValues()
// 	} else {
// 		cfg = Config.Copy()
// 	}

// 	encode = cfg

// 	if p.Field != "" {
// 		encode, err = cfg.Get(p.Field)
// 		if err != nil {
// 			return fmt.Errorf("error getting %s from config: %s", p.Field, err)
// 		}
// 	}

// 	switch p.Format {
// 	case "json":
// 		if p.Concise {
// 			*res, err = json.Marshal(encode)
// 		} else {
// 			*res, err = json.MarshalIndent(encode, "", " ")
// 		}
// 	case "yaml":
// 		*res, err = yaml.Marshal(encode)
// 	}
// 	if err != nil {
// 		return fmt.Errorf("error getting config: %s", err)
// 	}

// 	return nil
// }

// // SetConfig validates and saves the config (passed in as `res`)
// func SetConfig(res *config.Config) error {
// 	if err := res.Validate(); err != nil {
// 		return fmt.Errorf("error validating config: %s", err)
// 	}
// 	if err := SaveConfig(); err != nil {
// 		return fmt.Errorf("error saving config: %s", err)
// 	}

// 	Config = res.WithPrivateValues(Config)

// 	return nil
// }

// NewConfigRequests creates a configuration handle from an instance
func NewConfigRequests(cfg *config.Config, setCfg func(*config.Config) error, cli *rpc.Client) *ConfigRequests {
	return &ConfigRequests{
		cfg:    cfg,
		setCfg: setCfg,
		cli:    cli,
	}
}

// ConfigRequests encapsulates changes to a qri configuration
type ConfigRequests struct {
	cfg    *config.Config
	setCfg func(*config.Config) error
	cli    *rpc.Client
}

// CoreRequestsName specifies this is a configuration handle
func (h ConfigRequests) CoreRequestsName() string { return "config" }

// GetConfigParams are the params needed to format/specify the fields in bytes
// returned from the GetConfig function
type GetConfigParams struct {
	Field          string
	WithPrivateKey bool
	Format         string
	Concise        bool
}

// GetConfig returns the Config, or one of the specified fields of the Config,
// as a slice of bytes the bytes can be formatted as json, concise json, or yaml
func (h ConfigRequests) GetConfig(p *GetConfigParams, res *[]byte) error {
	if h.cli != nil {
		return fmt.Errorf("GetConfig cannot be called over RPC")
	}

	var (
		err    error
		cfg    = h.cfg
		encode interface{}
	)

	if !p.WithPrivateKey {
		cfg = cfg.WithoutPrivateValues()
	} else {
		cfg = cfg.Copy()
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

// SetConfig validates, updates and saves the config
func (h ConfigRequests) SetConfig(update *config.Config, set *bool) (err error) {
	if h.cli != nil {
		return fmt.Errorf("SetConfig cannot be called over RPC")
	}

	if err = update.Validate(); err != nil {
		return fmt.Errorf("error validating config: %s", err)
	}

	cfg := update.WithPrivateValues(h.cfg)
	if err = h.setCfg(cfg); err != nil {
		return
	}

	*set = true
	return nil
}
