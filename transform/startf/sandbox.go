package startf

import (
	"fmt"
	"net/http"
	"strings"

	starhttp "github.com/qri-io/starlib/http"
	"github.com/qri-io/starlib/util"
	"go.starlark.net/starlark"
)

var (
	httpGuard = &HTTPGuard{NetworkEnabled: true}
	// ErrNtwkDisabled is returned whenever a network call is attempted but h.NetworkEnabled is false
	ErrNtwkDisabled = fmt.Errorf("network use is disabled. http can only be used during download step")
)

// HTTPGuard protects network requests, only allowing when network is enabled
type HTTPGuard struct {
	NetworkEnabled bool
}

// Allowed implements starlib/http RequestGuard
func (h *HTTPGuard) Allowed(req *http.Request) error {
	if !h.NetworkEnabled {
		return ErrNtwkDisabled
	}
	return nil
}

// EnableNtwk allows network calls
func (h *HTTPGuard) EnableNtwk() {
	h.NetworkEnabled = true
}

// DisableNtwk prevents network calls from succeeding
func (h *HTTPGuard) DisableNtwk() {
	h.NetworkEnabled = false
}

func init() {
	// connect httpGuard instance to starlib http guard
	starhttp.Guard = httpGuard
}

type config map[string]interface{}

var (
	_ starlark.Value    = (*config)(nil)
	_ starlark.HasAttrs = (*config)(nil)
)

func (c config) Type() string          { return "config" }
func (c config) String() string        { return mapStringRepr(c) }
func (c config) Freeze()               {} // noop
func (c config) Truth() starlark.Bool  { return starlark.True }
func (c config) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", c.Type()) }

func (c config) AttrNames() []string { return []string{"get"} }
func (c config) Attr(s string) (starlark.Value, error) {
	if s == "get" {
		return starlark.NewBuiltin("get", configGet).BindReceiver(c), nil
	}
	return nil, nil
}

func configGet(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		self = b.Receiver().(config)
		key  string
		def  starlark.Value
	)
	if err := starlark.UnpackPositionalArgs("get", args, kwargs, 1, &key, &def); err != nil {
		return starlark.None, err
	}

	v, ok := self[key]
	if !ok {
		if def != nil {
			return def, nil
		}
		return starlark.None, nil
	}
	return util.Marshal(v)
}

type secrets = config

func mapStringRepr(m map[string]interface{}) string {
	builder := strings.Builder{}
	builder.WriteString("{ ")
	i := 0
	for k, v := range m {
		i++
		if i == len(m) {
			builder.WriteString(fmt.Sprintf("%s: %v ", k, v))
			break
		}
		builder.WriteString(fmt.Sprintf("%s: %v, ", k, v))
	}
	builder.WriteString("}")
	return builder.String()
}
