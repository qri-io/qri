package handlers

import (
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/logging"
	"github.com/qri-io/qri/repo"
)

// HistoryHandlers wraps a HistoryRequests with http.HandlerFuncs
type HistoryHandlers struct {
	core.HistoryRequests
	log logging.Logger
}

func NewHistoryHandlers(log logging.Logger, r repo.Repo) *HistoryHandlers {
	req := core.NewHistoryRequests(r, nil)
	h := HistoryHandlers{*req, log}
	return &h
}

func (h *HistoryHandlers) LogHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.logHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *HistoryHandlers) logHandler(w http.ResponseWriter, r *http.Request) {
	params := &core.LogParams{
		ListParams: core.ListParamsFromRequest(r),
		Path:       datastore.NewKey(r.URL.Path[len("/history/"):]),
	}

	res := []*repo.DatasetRef{}
	if err := h.Log(params, &res); err != nil {
		h.log.Infof("")
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WritePageResponse(w, res, r, params.Page())
}
