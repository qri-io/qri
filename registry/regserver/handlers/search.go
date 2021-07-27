package handlers

import (
	"encoding/json"
	"net/http"

	apiutil "github.com/qri-io/qri/api/util"
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
		if r.Header.Get("Content-Type") == "application/json" && r.Method == "POST" {
			if err := json.NewDecoder(r.Body).Decode(p); err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			if p.Limit == 0 {
				p.Limit = defaultLimit
			}
		} else {
			p.Limit = apiutil.ReqParamInt(r, "limit", defaultLimit)
			p.Offset = apiutil.ReqParamInt(r, "offset", defaultOffset)
			p.Q = r.FormValue("q")
		}
		results, err := s.Search(*p)
		if err != nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		apiutil.WriteResponse(w, results)
	}
}
