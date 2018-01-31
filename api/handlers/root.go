package handlers

import (
  "errors"
  "net/http"

  util "github.com/datatogether/api/apiutil"
  "github.com/qri-io/qri/core"
  "github.com/qri-io/qri/repo/profile"
)

type RootHandler struct {
  dsh *DatasetHandlers
  ph  *PeerHandlers
}

// WebappHandler renders the home page
func WebappHandler(w http.ResponseWriter, r *http.Request) {
  renderTemplate(w, "webapp")
}

func NewRootHandlers(dsh *DatasetHandlers, ph *PeerHandlers) *RootHandler {
  return &RootHandler{dsh, ph}
}

func (mh *RootHandler) RootHandler(w http.ResponseWriter, r *http.Request) {
  ref := DatasetRefFromCtx(r.Context())
  if ref == nil {
    WebappHandler(w, r)
    return
  }
  if ref.IsPeerRef() {
    p := &core.PeerInfoParams{
      Peername: ref.Peername,
    }
    res := &profile.Profile{}
    err := mh.ph.Info(p, res)
    if err != nil {
      util.WriteErrResponse(w, http.StatusInternalServerError, err)
      return
    }
    if res.ID == "" {
      util.WriteErrResponse(w, http.StatusNotFound, errors.New("cannot find peer"))
      return
    }
    util.WriteResponse(w, res)
    return
  } else {
    util.WriteErrResponse(w, http.StatusInternalServerError, errors.New("TBD"))
    return
  }
}
