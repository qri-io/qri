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

// Name returns the name of this method group
func (m *ConfigMethods) Name() string {
	return "config"
}

// Attributes defines attributes for each method
func (m *ConfigMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		// config methods are not allowed over HTTP nor RPC
		"getconfig":     {"", ""},
		"getconfigkeys": {"", ""},
		"setconfig":     {"", ""},
	}
}

// Config returns the `Config` that the instance has registered
func (inst *Instance) Config() *ConfigMethods {
	return &ConfigMethods{inst: inst}
}

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
	got, _, err := m.inst.Dispatch(ctx, dispatchMethodName(m, "getconfig"), p)
	if res, ok := got.([]byte); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// GetConfigKeys returns the Config key fields, or sub keys of the specified
// fields of the Config, as a slice of bytes to be used for auto completion
func (m *ConfigMethods) GetConfigKeys(ctx context.Context, p *GetConfigParams) ([]byte, error) {
	got, _, err := m.inst.Dispatch(ctx, dispatchMethodName(m, "getconfigkeys"), p)
	if res, ok := got.([]byte); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// SetConfig validates, updates and saves the config
func (m *ConfigMethods) SetConfig(ctx context.Context, update *config.Config) (*bool, error) {
	got, _, err := m.inst.Dispatch(ctx, dispatchMethodName(m, "setconfig"), update)
	if res, ok := got.(*bool); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// configImpl holds the method implementations for ConfigMethod
type configImpl struct{}

// GetConfig returns the Config, or one of the specified fields of the Config
func (configImpl) GetConfig(scope scope, p *GetConfigParams) ([]byte, error) {
	var (
		cfg    = scope.Config()
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

// GetConfigKeys returns the Config key fields, or sub keys of the
func (configImpl) GetConfigKeys(scope scope, p *GetConfigParams) ([]byte, error) {
	var (
		cfg    = scope.Config()
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

// SetConfig validates, updates, and saves the config
func (configImpl) SetConfig(scope scope, update *config.Config) (*bool, error) {
	res := false
	if err := update.Validate(); err != nil {
		return &res, fmt.Errorf("validating config: %w", err)
	}

	if err := scope.ChangeConfig(update); err != nil {
		return &res, err
	}
	res = true
	return &res, nil
}
