package api

import (
	"fmt"
	"net/http"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// LogHandlers wraps a LogRequests with http.HandlerFuncs
type LogHandlers struct {
	lib.LogRequests
}

// NewLogHandlers allocates a LogHandlers pointer
func NewLogHandlers(n *p2p.QriNode) *LogHandlers {
	req := lib.NewLogRequests(n, nil)
	h := LogHandlers{*req}
	return &h
}

// LogHandler is the endpoint for dataset logs
func (h *LogHandlers) LogHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.logHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

type DatasetLogItem = base.DatasetLogItem

func (h *LogHandlers) logHandler(w http.ResponseWriter, r *http.Request) {
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
		Ref:        args.String(),
		ListParams: lp,
	}

	res := []DatasetLogItem{}
	if err := h.Log(params, &res); err != nil {
		if err == repo.ErrNoHistory {
			util.WriteErrResponse(w, http.StatusUnprocessableEntity, err)
			return
		}
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := util.WritePageResponse(w, res, r, params.Page()); err != nil {
		log.Infof("error list dataset history response: %s", err.Error())
	}
}
