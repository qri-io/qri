package api

import (
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/lib"
)

// ConfigHandlers wraps a ConfigMethods with http.HandlerFuncs
type ConfigHandlers struct {
	cm       lib.ConfigMethods
	readOnly bool
}

// NewConfigHandlers allocates a ConfigHandlers pointer
func NewConfigHandlers(inst *lib.Instance) *ConfigHandlers {
	req := lib.NewConfigMethods(inst)
	h := ConfigHandlers{
		cm: *req,
	}
	return &h
}

// ConfigHandler is the endpoint for configs
func (h *ConfigHandlers) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.configHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// ConfigKeysHandler is the endpoint for config keys
func (h *ConfigHandlers) ConfigKeysHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.configKeysHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *ConfigHandlers) configHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.GetConfigParams{}
	err := UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.cm.GetConfig(r.Context(), params)
	if err != nil {
		log.Infof("error getting config: %s", err.Error())
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	w.Write(res)
}

func (h *ConfigHandlers) configKeysHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.GetConfigParams{}
	err := UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.cm.GetConfigKeys(r.Context(), params)
	if err != nil {
		log.Infof("error getting config: %s", err.Error())
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	w.Write(res)
}
