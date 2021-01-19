package context

import (
	"fmt"

	"github.com/qri-io/starlib/util"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Context carries values across function calls in a transformation
type Context struct {
	// Results carries the return values of special function calls
	results starlark.StringDict
	values  starlark.StringDict
	config  map[string]interface{}
	secrets map[string]interface{}
}

// NewContext creates a new contex
func NewContext(config, secrets map[string]interface{}) *Context {
	return &Context{
		results: starlark.StringDict{},
		values:  starlark.StringDict{},
		config:  config,
		secrets: secrets,
	}
}

// Struct delivers this context as a starlark struct
func (c *Context) Struct() *starlarkstruct.Struct {
	dict := starlark.StringDict{
		"set":        starlark.NewBuiltin("set", c.setValue),
		"get":        starlark.NewBuiltin("get", c.getValue),
		"get_config": starlark.NewBuiltin("get_config", c.GetConfig),
		"get_secret": starlark.NewBuiltin("get_secret", c.GetSecret),
	}

	for k, v := range c.results {
		dict[k] = v
	}

	return starlarkstruct.FromStringDict(starlark.String("context"), dict)
}

// SetResult places the result of a function call in the results stringDict
// any results set here will be placed in the context struct field by name
func (c *Context) SetResult(name string, value starlark.Value) {
	c.results[name] = value
}

func (c *Context) setValue(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		key   starlark.String
		value starlark.Value
	)
	if err := starlark.UnpackArgs("set", args, kwargs, "key", &key, "value", &value); err != nil {
		return starlark.None, err
	}

	c.values[string(key)] = value
	return starlark.None, nil
}

func (c *Context) getValue(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key starlark.String
	if err := starlark.UnpackArgs("get", args, kwargs, "key", &key); err != nil {
		return starlark.None, err
	}
	if v, ok := c.values[string(key)]; ok {
		return v, nil
	}
	return starlark.None, fmt.Errorf("value %s not set in context", string(key))
}

// GetSecret fetches a secret for a given string
func (c *Context) GetSecret(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if c.secrets == nil {
		return starlark.None, fmt.Errorf("no secrets provided")
	}

	var key starlark.String
	if err := starlark.UnpackPositionalArgs("get_secret", args, kwargs, 1, &key); err != nil {
		return nil, err
	}

	return util.Marshal(c.secrets[string(key)])
}

// GetConfig returns transformation configuration details
// TODO - supplying a string argument to qri.get_config('foo') should return the single config value instead of the whole map
func (c *Context) GetConfig(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if c.config == nil {
		return starlark.None, fmt.Errorf("no config provided")
	}

	var key starlark.String
	if err := starlark.UnpackPositionalArgs("get_config", args, kwargs, 1, &key); err != nil {
		return nil, err
	}

	return util.Marshal(c.config[string(key)])
}
