package handlers

import (
	"encoding/json"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/logging"
	"github.com/qri-io/qri/repo"
)

// ProfileHandlers wraps a requests struct to interface with http.HandlerFunc
type ProfileHandlers struct {
	core.ProfileRequests
	log logging.Logger
}

func NewProfileHandlers(log logging.Logger, r repo.Repo) *ProfileHandlers {
	req := core.NewProfileRequests(r, nil)
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
	res := &core.Profile{}
	if err := h.GetProfile(&args, res); err != nil {
		h.log.Infof("error getting profile: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h *ProfileHandlers) saveProfileHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.Profile{}
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	res := &core.Profile{}
	if err := h.SaveProfile(p, res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

func (h *ProfileHandlers) SetProfilePhotoHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "PUT", "POST":
		h.setProfilePhotoHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *ProfileHandlers) setProfilePhotoHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.FileParams{}
	if r.Header.Get("Content-Type") == "application/json" {
		json.NewDecoder(r.Body).Decode(p)
	} else {
		infile, header, err := r.FormFile("file")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		p = &core.FileParams{
			// Url:          r.FormValue("url"),
			Filename: header.Filename,
			Data:     infile,
		}
	}

	res := &core.Profile{}
	if err := h.SetProfilePhoto(p, res); err != nil {
		h.log.Infof("error initializing dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

func (h *ProfileHandlers) SetPosterHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "PUT", "POST":
		h.setPosterHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *ProfileHandlers) setPosterHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.FileParams{}
	if r.Header.Get("Content-Type") == "application/json" {
		json.NewDecoder(r.Body).Decode(p)
	} else {
		infile, header, err := r.FormFile("file")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		p = &core.FileParams{
			Filename: header.Filename,
			Data:     infile,
		}
	}

	res := &core.Profile{}
	if err := h.SetPosterPhoto(p, res); err != nil {
		h.log.Infof("error initializing dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}
