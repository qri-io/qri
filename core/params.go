package core

import (
	"net/http"

	util "github.com/datatogether/api/apiutil"
)

const DEFAULT_PAGE_SIZE = 100
const DEFAULT_LIST_ORDERING = "created"

var validOrderingArguments = map[string]bool{
	"created": true,
	//"modified": true,
	//"name": true,
}

type GetParams struct {
	Username string
	Name     string
	Hash     string
}

type ListParams struct {
	OrderBy string
	Limit   int
	Offset  int
}

// ListParamsFromRequest extracts and returns a ListParams struct
// given a pointer to an http request *r
func ListParamsFromRequest(r *http.Request) ListParams {
	var lp ListParams
	var pageIndex int
	// Limit
	if i, err := util.ReqParamInt("pageSize", r); err == nil {
		lp.Limit = i
	}
	if lp.Limit <= 0 {
		lp.Limit = DEFAULT_PAGE_SIZE
	}
	// Offset
	if i, err := util.ReqParamInt("page", r); err == nil {
		pageIndex = i
	}
	if pageIndex < 0 {
		pageIndex = 0
	}
	lp.Offset = pageIndex * lp.Limit
	// Orderby
	orderKey := r.FormValue("orderBy")
	if _, ok := validOrderingArguments[orderKey]; ok {
		lp.OrderBy = orderKey
	} else {
		lp.OrderBy = DEFAULT_LIST_ORDERING
	}
	return lp

}
