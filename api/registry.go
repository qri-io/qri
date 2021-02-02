package api

import (
	"encoding/json"
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/lib"
)

// RegistryClientHandlers wraps a requests struct to interface with http.HandlerFunc
type RegistryClientHandlers struct {
	*lib.RegistryClientMethods
	readOnly bool
}

// NewRegistryClientHandlers allocates a RegistryClientHandlers pointer
func NewRegistryClientHandlers(inst *lib.Instance, readOnly bool) *RegistryClientHandlers {
	h := &RegistryClientHandlers{
		RegistryClientMethods: lib.NewRegistryClientMethods(inst),
		readOnly:              readOnly,
	}
	return h
}

// CreateProfileHandler creates a profile, associating it with a private key
func (h *RegistryClientHandlers) CreateProfileHandler(w http.ResponseWriter, r *http.Request) {
	if h.readOnly {
		readOnlyResponse(w, "/registry/profiles/new")
		return
	}

	if r.Method != http.MethodPost {
		util.NotFoundHandler(w, r)
		return
	}

	p := &lib.RegistryProfile{}
	ok := false
	err := json.NewDecoder(r.Body).Decode(p)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	err = h.CreateProfile(p, &ok)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, p)
}

// ProveProfileKeyHandler proves a user controls both a registry profile and a
// new keypair
func (h *RegistryClientHandlers) ProveProfileKeyHandler(w http.ResponseWriter, r *http.Request) {
	if h.readOnly {
		readOnlyResponse(w, "/registry/profiles/prove")
	}

	if r.Method != http.MethodPost {
		util.NotFoundHandler(w, r)
		return
	}

	p := &lib.RegistryProfile{}
	ok := false
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	if err := h.ProveProfileKey(p, &ok); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, p)
}
