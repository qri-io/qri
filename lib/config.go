package lib

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/qri-io/qri/config"
)

// ConfigMethods defines functions for working with with Qri configuration
// details.
type ConfigMethods interface {
	Methods
	GetConfig(p *GetConfigParams, res *[]byte) error
	SetConfig(update *config.Config, set *bool) error
}

// NewConfigMethods creates a configuration handle from an instance
func NewConfigMethods(inst Instance) ConfigMethods {
	return configHandle{inst: inst}
}

// GetConfigParams are the params needed to format/specify the fields in bytes
// returned from the GetConfig function
type GetConfigParams struct {
	Field          string
	WithPrivateKey bool
	Format         string
	Concise        bool
}

type configHandle struct {
	inst Instance
}

// MethodsKind specifies this is a configuration handle
func (h configHandle) MethodsKind() string { return "ConfigMethods" }

// GetConfig returns the Config, or one of the specified fields of the Config,
// as a slice of bytes the bytes can be formatted as json, concise json, or yaml
func (h configHandle) GetConfig(p *GetConfigParams, res *[]byte) error {
	var (
		err    error
		cfg    = &config.Config{}
		encode interface{}
	)

	if !p.WithPrivateKey {
		cfg = h.inst.Config().WithoutPrivateValues()
	} else {
		cfg = h.inst.Config().Copy()
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
func (h configHandle) SetConfig(update *config.Config, set *bool) (err error) {
	if err = update.Validate(); err != nil {
		return fmt.Errorf("error validating config: %s", err)
	}

	writable, ok := h.inst.(WritableInstance)
	if !ok {
		return ErrNotWritable
	}

	cfg := update.WithPrivateValues(h.inst.Config())
	if err = writable.SetConfig(cfg); err != nil {
		return
	}

	*set = true
	return nil
}
