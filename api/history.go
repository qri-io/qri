package api

import (
	"net/http"

	"fmt"
	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
)

// HistoryHandlers wraps a HistoryRequests with http.HandlerFuncs
type HistoryHandlers struct {
	lib.HistoryRequests
	repo repo.Repo
}

// NewHistoryHandlers allocates a HistoryHandlers pointer
func NewHistoryHandlers(r repo.Repo) *HistoryHandlers {
	req := lib.NewHistoryRequests(r, nil)
	h := HistoryHandlers{*req, r}
	return &h
}

// LogHandler is the endpoint for dataset logs
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
	args, err := DatasetRefFromPath(r.URL.Path[len("/history"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	if args.Name == "" && args.Path == "" {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("name of dataset or path needed"))
		return
	}

	lp := lib.ListParamsFromRequest(r)
	lp.Peername = args.Peername

	params := &lib.LogParams{
		ListParams: lp,
		Ref:        args,
	}

	res := []repo.DatasetRef{}
	if err := h.Log(params, &res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WritePageResponse(w, res, r, params.Page())
}
