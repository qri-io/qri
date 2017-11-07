package core

import (
	"net/http"

	util "github.com/datatogether/api/apiutil"
)

const DEFAULT_PAGE_SIZE = 100

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

// ListParamsFromRequest extracts ListParams from an http.Request pointer
func ListParamsFromRequest(r *http.Request) ListParams {
	var lp ListParams
	var pageIndex int
	if i, err := util.ReqParamInt("pageSize", r); err == nil {
		lp.Limit = i
	}
	if lp.Limit <= 0 {
		lp.Limit = DEFAULT_PAGE_SIZE
	}
	if i, err := util.ReqParamInt("page", r); err == nil {
		pageIndex = i
	}
	if pageIndex < 0 {
		pageIndex = 0
	}
	lp.Offset = pageIndex * lp.Limit
	// lp.OrderBy defaults to empty string
	return lp

}

// Page converts a ListParams struct to a util.Page struct
func (lp ListParams) Page() util.Page {
	var number, size int
	size = lp.Limit
	if size <= 0 {
		size = DEFAULT_PAGE_SIZE
	}
	number = lp.Offset/size + 1
	return util.NewPage(number, size)
}
