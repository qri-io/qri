package handlers

import (
	"encoding/json"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/logging"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// ProfileHandlers wraps a requests struct to interface with http.HandlerFunc
type ProfileHandlers struct {
	core.ProfileRequests
	log logging.Logger
}

func NewProfileHandlers(log logging.Logger, r repo.Repo) *ProfileHandlers {
	req := core.NewProfileRequests(r)
	h := ProfileHandlers{*req, log}
	return &h
}

func (h *ProfileHandlers) ProfileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.getProfileHandler(w, r)
	case "POST":
		h.saveProfileHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *ProfileHandlers) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	args := true
	res := &profile.Profile{}
	if err := h.GetProfile(&args, res); err != nil {
		h.log.Infof("error getting profile: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h *ProfileHandlers) saveProfileHandler(w http.ResponseWriter, r *http.Request) {
	p := &profile.Profile{}
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	res := &profile.Profile{}
	if err := h.SaveProfile(p, res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}
