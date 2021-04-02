package startf

import (
	"fmt"
	"net/http"

	starhttp "github.com/qri-io/starlib/http"
)

var (
	httpGuard = &HTTPGuard{}
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
