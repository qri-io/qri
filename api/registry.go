package api

import (
	"net/http"

	"github.com/qri-io/qri/lib"
)

// RegistryClientHandlers wraps a requests struct to interface with http.HandlerFunc
type RegistryClientHandlers struct {
	lib.RegistryClientMethods
}

// NewRegistryClientHandlers allocates a RegistryClientHandlers pointer
func NewRegistryClientHandlers(inst *lib.Instance) *RegistryClientHandlers {
	m := lib.RegistryClientMethods(*inst)
	h := RegistryClientHandlers{m}
	return &h
}

// CreateProfileHandler creates a profile, associating it with a private key
func (h *RegistryClientHandlers) CreateProfileHandler(w http.ResponseWriter, r *http.Request) {
	// TODO (b5)
}

// ProveProfileKeyHandler proves a user controls both a registry profile and a
// new keypair
func (h *RegistryClientHandlers) ProveProfileKeyHandler(w http.ResponseWriter, r *http.Request) {
	// TODO (b5)
}
