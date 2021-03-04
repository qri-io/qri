package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/qri-io/qri/config"
)

// ConfigMethods encapsulates changes to a qri configuration
type ConfigMethods struct {
	inst *Instance
}

// NewConfigMethods creates a configuration handle from an instance
func NewConfigMethods(inst *Instance) *ConfigMethods {
	return &ConfigMethods{inst: inst}
}

// CoreRequestsName specifies this is a configuration handle
func (m ConfigMethods) CoreRequestsName() string { return "config" }

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
func (m *ConfigMethods) GetConfig(ctx context.Context, p *GetConfigParams) ([]byte, error) {
	if m.inst.http != nil {
		return nil, ErrUnsupportedRPC
	}

	var (
		cfg    = m.inst.cfg
		encode interface{}
		err    error
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
			return nil, fmt.Errorf("error getting %s from config: %w", p.Field, err)
		}
	}

	var res []byte

	switch p.Format {
	case "json":
		if p.Concise {
			res, err = json.Marshal(encode)
		} else {
			res, err = json.MarshalIndent(encode, "", " ")
		}
	case "yaml":
		res, err = yaml.Marshal(encode)
	}
	if err != nil {
		return nil, fmt.Errorf("error getting config: %w", err)
	}

	return res, nil
}

// GetConfigKeys returns the Config key fields, or sub keys of the specified
// fields of the Config, as a slice of bytes to be used for auto completion
func (m *ConfigMethods) GetConfigKeys(ctx context.Context, p *GetConfigParams) ([]byte, error) {
	if m.inst.http != nil {
		return nil, ErrUnsupportedRPC
	}

	var (
		cfg    = m.inst.cfg
		encode interface{}
		err    error
	)

	cfg = cfg.WithoutPrivateValues()

	encode = cfg
	keyPrefix := ""

	if len(p.Field) > 0 && p.Field[len(p.Field)-1] == '.' {
		p.Field = p.Field[:len(p.Field)-1]
	}
	parentKey := p.Field

	if p.Field != "" {
		fieldArgs := strings.Split(p.Field, ".")
		encode, err = cfg.Get(p.Field)
		if err != nil {
			keyPrefix = fieldArgs[len(fieldArgs)-1]
			if len(fieldArgs) == 1 {
				encode = cfg
				parentKey = ""
			} else {
				parentKey = strings.Join(fieldArgs[:len(fieldArgs)-1], ".")
				newEncode, fieldErr := cfg.Get(parentKey)
				if fieldErr != nil {
					return nil, fmt.Errorf("error getting %s from config: %w", p.Field, err)
				}
				encode = newEncode
			}
		}
	}

	return parseKeys(encode, keyPrefix, parentKey)
}

func parseKeys(cfg interface{}, prefix, parentKey string) ([]byte, error) {
	cfgBytes, parseErr := json.Marshal(cfg)
	if parseErr != nil {
		return nil, parseErr
	}

	cfgMap := map[string]interface{}{}
	parseErr = json.Unmarshal(cfgBytes, &cfgMap)
	if parseErr != nil {
		return nil, parseErr
	}

	buff := bytes.Buffer{}
	for s := range cfgMap {
		if prefix != "" && !strings.HasPrefix(s, prefix) {
			continue
		}
		if parentKey != "" {
			buff.WriteString(parentKey)
			buff.WriteString(".")
		}
		buff.WriteString(s)
		buff.WriteString("\n")
	}
	if len(buff.Bytes()) > 0 {
		return buff.Bytes(), nil
	}
	return nil, fmt.Errorf("error getting %s from config", prefix)
}

// SetConfig validates, updates and saves the config
func (m *ConfigMethods) SetConfig(ctx context.Context, update *config.Config) (*bool, error) {
	res := false
	if m.inst.http != nil {
		return &res, ErrUnsupportedRPC
	}

	if err := update.Validate(); err != nil {
		return &res, fmt.Errorf("validating config: %w", err)
	}

	if err := m.inst.ChangeConfig(update); err != nil {
		return &res, err
	}
	res = true
	return &res, nil
}
