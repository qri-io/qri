package queries

import (
	"encoding/json"
	"fmt"
	util "github.com/datatogether/api/apiutil"
	// "github.com/ipfs/go-datastore"
	// "github.com/qri-io/castore"
	"github.com/qri-io/dataset"
	"net/http"
)

func NewHandlers(r Requests) *Handlers {
	return &Handlers{r}
}

// Handlers wraps a requests struct to interface with http.HandlerFunc
type Handlers struct {
	Requests
}

// func (d *Handlers) ListHandler(w http.ResponseWriter, r *http.Request) {
// 	switch r.Method {
// 	case "OPTIONS":
// 		util.EmptyOkHandler(w, r)
// 	case "GET":
// 		d.listQueriesHandler(w, r)
// 	default:
// 		util.NotFoundHandler(w, r)
// 	}
// }

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

// func (d *Handlers) listQueriesHandler(w http.ResponseWriter, r *http.Request) {
// 	p := util.PageFromRequest(r)
// 	res := map[string]datastore.Key{}
// 	args := &ListParams{
// 		Limit:   p.Limit(),
// 		Offset:  p.Offset(),
// 		OrderBy: "created",
// 	}
// 	err := d.List(args, &res)
// 	if err != nil {
// 		util.WriteErrResponse(w, http.StatusInternalServerError, err)
// 		return
// 	}
// 	util.WritePageResponse(w, res, r, p)
// }

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
	p := &dataset.Query{}
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res := &dataset.Dataset{}
	if err := h.Run(p, res); err != nil {
		fmt.Println("err:")
		fmt.Println(err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	fmt.Println("response:")
	fmt.Println(res)
	util.WriteResponse(w, res)
}
