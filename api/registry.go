package api

import (
	"encoding/json"
	"net/http"
	"strings"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/dsref"
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

	if r.Method != "POST" {
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

	if r.Method != "POST" {
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

// HomeHandler fetches an index of content from the registry
func (h *RegistryClientHandlers) HomeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.homeHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *RegistryClientHandlers) homeHandler(w http.ResponseWriter, r *http.Request) {
	res := map[string][]dsref.VersionInfo{}
	p := false
	if err := h.Home(&p, &res); err != nil {
		log.Infof("home error: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

// DatasetPreviewHandler fetches a dataset preview from the registry
func (h *RegistryClientHandlers) DatasetPreviewHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.previewHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *RegistryClientHandlers) previewHandler(w http.ResponseWriter, r *http.Request) {
	refstr := strings.TrimPrefix(r.URL.Path, "/registry/dataset/preview/")
	res := &dataset.Dataset{}
	if err := h.Preview(&refstr, res); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	util.WriteResponse(w, res)
}
