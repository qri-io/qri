package queries

import (
	"encoding/json"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/server/logging"
)

func NewHandlers(log logging.Logger, store cafs.Filestore, r repo.Repo) *Handlers {
	req := NewRequests(store, r)
	return &Handlers{*req, log}
}

// Handlers wraps a requests struct to interface with http.HandlerFunc
type Handlers struct {
	Requests
	log logging.Logger
}

func (d *Handlers) ListHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		d.listQueriesHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// func (d *Handlers) GetHandler(w http.ResponseWriter, r *http.Request) {
// 	switch r.Method {
// 	case "OPTIONS":
// 		util.EmptyOkHandler(w, r)
// 	case "GET":
// 		d.getDatasetHandler(w, r)
// 	default:
// 		util.NotFoundHandler(w, r)
// 	}
// }

// func (d *Handlers) getDatasetHandler(w http.ResponseWriter, r *http.Request) {
// 	res := &dataset.Dataset{}
// 	args := &GetParams{
// 		Path: r.URL.Path[len("/queries/"):],
// 		Hash: r.FormValue("hash"),
// 	}
// 	err := d.Get(args, res)
// 	if err != nil {
// 		util.WriteErrResponse(w, http.StatusInternalServerError, err)
// 		return
// 	}
// 	util.WriteResponse(w, res)
// }

func (h *Handlers) listQueriesHandler(w http.ResponseWriter, r *http.Request) {
	p := util.PageFromRequest(r)
	res := []*repo.DatasetRef{}
	args := &ListParams{
		Limit:   p.Limit(),
		Offset:  p.Offset(),
		OrderBy: "created",
	}
	err := h.List(args, &res)
	if err != nil {
		h.log.Infof("error listing datasets: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WritePageResponse(w, res, r, p)
}

func (h *Handlers) RunHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST":
		h.runHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *Handlers) runHandler(w http.ResponseWriter, r *http.Request) {
	ds := &dataset.Dataset{}
	if err := json.NewDecoder(r.Body).Decode(ds); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	format := r.FormValue("format")
	if format == "" {
		format = "csv"
	}
	df, err := dataset.ParseDataFormatString(format)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	p := &RunParams{
		SaveName: r.FormValue("name"),
		Dataset:  ds,
	}
	p.Format = df

	res := &repo.DatasetRef{}
	if err := h.Run(p, res); err != nil {
		h.log.Infof("error running query: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}
