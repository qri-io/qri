package lib

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/qri-io/qri/config"
)

// ConfigMethods encapsulates changes to a qri configuration
type ConfigMethods interface {
	// CoreRequestsName specifies participation in the methods interface
	// TODO (b5): update func name when change happens
	CoreRequestsName() string
	// GetConfig returns the Config, or one of the specified fields of the Config,
	// as a slice of bytes the bytes can be formatted as json, concise json, or yaml
	GetConfig(p *GetConfigParams, res *[]byte) (err error)
	// SetConfig validates, updates and saves the config
	SetConfig(update *config.Config, set *bool) (err error)
}

// NewConfigMethods creates a configuration handle from an instance
func NewConfigMethods(inst *Instance) ConfigMethods {
	return configMethods{inst: inst}
}

// configMethods is the private implementation of ConfigMethods
type configMethods struct {
	inst *Instance
}

// CoreRequestsName specifies this is a configuration handle
func (m configMethods) CoreRequestsName() string { return "config" }

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
func (m configMethods) GetConfig(p *GetConfigParams, res *[]byte) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("ConfigMethods.GetConfig", p, res)
	}

	var (
		cfg    = m.inst.cfg
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
func (m configMethods) SetConfig(update *config.Config, set *bool) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("ConfigMethods.SetConfig", update, set)
	}

	if err = update.Validate(); err != nil {
		return fmt.Errorf("error validating config: %s", err)
	}

	cfg := update.WithPrivateValues(m.inst.cfg)

	*set = true

	return m.inst.ChangeConfig(cfg)
}
