package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/qri-io/apiutil"
	"github.com/qri-io/qri/registry"
)

const (
	defaultOffset = 0
	defaultLimit  = 25
)

// NewSearchHandler creates a search handler function taht operates on a *registry.Searchable
func NewSearchHandler(s registry.Searchable) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := &registry.SearchParams{}
		switch r.Header.Get("Content-Type") {
		case "application/json":
			if err := json.NewDecoder(r.Body).Decode(p); err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			if p.Limit == 0 {
				p.Limit = defaultLimit
			}
		default:
			// read form values
			var err error
			if p.Limit, err = apiutil.ReqParamInt("limit", r); err != nil {
				p.Limit = defaultLimit
				err = nil
			}
			if p.Offset, err = apiutil.ReqParamInt("offset", r); err != nil {
				p.Offset = defaultOffset
				err = nil
			}
			p.Q = r.FormValue("q")
		}
		switch r.Method {
		case "GET":
			results, err := s.Search(*p)
			if err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			apiutil.WriteResponse(w, results)
			return
		}
	}
}
