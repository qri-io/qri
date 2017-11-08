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

// ListParams is the general input for any sort of Paginated Request
// ListParams define limits & offsets, not pages & page sizes.
// TODO - rename this to PageParams.
type ListParams struct {
	OrderBy string
	Limit   int
	Offset  int
}

// NewListParams creates a ListParams from page & pagesize, pages are 1-indexed
// (the first element is 1, not 0), NewListParams performs the conversion
func NewListParams(orderBy string, page, pageSize int) ListParams {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = DEFAULT_PAGE_SIZE
	}
	return ListParams{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}
}

// ListParamsFromRequest extracts ListParams from an http.Request pointer
func ListParamsFromRequest(r *http.Request) ListParams {
	var page, pageSize int
	if i, err := util.ReqParamInt("page", r); err == nil {
		page = i
	}
	if i, err := util.ReqParamInt("pageSize", r); err == nil {
		pageSize = i
	}
	return NewListParams(r.FormValue("orderBy"), page, pageSize)
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
