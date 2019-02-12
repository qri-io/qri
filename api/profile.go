package api

import (
	"encoding/json"
	"net/http"

	"fmt"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
)

// ProfileHandlers wraps a requests struct to interface with http.HandlerFunc
type ProfileHandlers struct {
	lib.ProfileRequests
	ReadOnly bool
}

// NewProfileHandlers allocates a ProfileHandlers pointer
func NewProfileHandlers(node *p2p.QriNode, readOnly bool) *ProfileHandlers {
	// TODO (b5): plumb config nonsense into here
	req := lib.NewProfileRequests(node, nil, "", nil)
	h := ProfileHandlers{*req, readOnly}
	return &h
}

// ProfileHandler is the endpoint for this peer's profile
func (h *ProfileHandlers) ProfileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		if h.ReadOnly {
			readOnlyResponse(w, "/profile' or '/me")
			return
		}
		h.getProfileHandler(w, r)
	case "POST":
		h.saveProfileHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *ProfileHandlers) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	args := true
	res := &config.ProfilePod{}
	if err := h.GetProfile(&args, res); err != nil {
		log.Infof("error getting profile: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h *ProfileHandlers) saveProfileHandler(w http.ResponseWriter, r *http.Request) {
	p := &config.ProfilePod{}
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error decoding request body: %s", err.Error()))
		return
	}
	res := &config.ProfilePod{}
	if err := h.SaveProfile(p, res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error saving profile: %s", err.Error()))
		return
	}
	util.WriteResponse(w, res)
}

// ProfilePhotoHandler is the endpoint for uploading this peer's profile photo
func (h *ProfileHandlers) ProfilePhotoHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.getProfilePhotoHandler(w, r)
	case "PUT", "POST":
		h.setProfilePhotoHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *ProfileHandlers) getProfilePhotoHandler(w http.ResponseWriter, r *http.Request) {
	data := []byte{}
	req := &config.ProfilePod{}
	req.Peername = r.FormValue("peername")
	req.ID = r.FormValue("id")

	if err := h.ProfilePhoto(req, &data); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(data)
}

func (h *ProfileHandlers) setProfilePhotoHandler(w http.ResponseWriter, r *http.Request) {
	p := &lib.FileParams{}
	if r.Header.Get("Content-Type") == "application/json" {
		json.NewDecoder(r.Body).Decode(p)
	} else {
		infile, header, err := r.FormFile("file")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		p = &lib.FileParams{
			// Url:          r.FormValue("url"),
			Filename: header.Filename,
			Data:     infile,
		}
	}

	res := &config.ProfilePod{}
	if err := h.SetProfilePhoto(p, res); err != nil {
		log.Infof("error initializing dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

// PosterHandler is the endpoint for uploading this peer's poster photo
func (h *ProfileHandlers) PosterHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.getPosterHandler(w, r)
	case "PUT", "POST":
		h.setPosterHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *ProfileHandlers) getPosterHandler(w http.ResponseWriter, r *http.Request) {
	data := []byte{}
	req := &config.ProfilePod{}
	req.Peername = r.FormValue("peername")
	req.ID = r.FormValue("id")

	if err := h.PosterPhoto(req, &data); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(data)
}

func (h *ProfileHandlers) setPosterHandler(w http.ResponseWriter, r *http.Request) {
	p := &lib.FileParams{}
	if r.Header.Get("Content-Type") == "application/json" {
		json.NewDecoder(r.Body).Decode(p)
	} else {
		infile, header, err := r.FormFile("file")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		p = &lib.FileParams{
			Filename: header.Filename,
			Data:     infile,
		}
	}

	res := &config.ProfilePod{}
	if err := h.SetPosterPhoto(p, res); err != nil {
		log.Infof("error initializing dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}
