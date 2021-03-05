package api

import (
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
)

// ProfileHandlers wraps a requests struct to interface with http.HandlerFunc
type ProfileHandlers struct {
	lib.ProfileMethods
	ReadOnly bool
}

// NewProfileHandlers allocates a ProfileHandlers pointer
func NewProfileHandlers(inst *lib.Instance, readOnly bool) *ProfileHandlers {
	h := ProfileHandlers{
		ProfileMethods: *lib.NewProfileMethods(inst),
		ReadOnly:       readOnly,
	}
	return &h
}

// ProfileHandler is the endpoint for this peer's profile
func (h *ProfileHandlers) ProfileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if h.ReadOnly {
			readOnlyResponse(w, "/profile' or '/me")
			return
		}
		h.getProfileHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *ProfileHandlers) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	args := true
	res, err := h.GetProfile(r.Context(), &args)
	if err != nil {
		log.Infof("error getting profile: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

// ProfilePhotoHandler is the endpoint for uploading this peer's profile photo
func (h *ProfileHandlers) ProfilePhotoHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.getProfilePhotoHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *ProfileHandlers) getProfilePhotoHandler(w http.ResponseWriter, r *http.Request) {
	params := &config.ProfilePod{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.ProfilePhoto(r.Context(), params)
	if err != nil {
		log.Infof("error getting profile photo: %s", err.Error())
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(res)
}

// PosterHandler is the endpoint for uploading this peer's poster photo
func (h *ProfileHandlers) PosterHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.getPosterHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *ProfileHandlers) getPosterHandler(w http.ResponseWriter, r *http.Request) {
	params := &config.ProfilePod{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.PosterPhoto(r.Context(), params)
	if err != nil {
		log.Infof("error getting profile poster: %s", err.Error())
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(res)
}
