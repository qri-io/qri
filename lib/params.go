package lib

import (
	"net/http"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/dag"
	"github.com/qri-io/qri/repo/profile"
)

// DefaultPageSize is the max number of items in a page if no
// Limit param is provided to a paginated method
const DefaultPageSize = 100

// ListParams is the general input for any sort of Paginated Request
// ListParams define limits & offsets, not pages & page sizes.
// TODO - rename this to PageParams.
type ListParams struct {
	ProfileID profile.ID
	Term      string
	Peername  string
	OrderBy   string
	Limit     int
	Offset    int
	// RPC is a horrible hack while we work to replace the net/rpc package
	// TODO - remove this
	RPC bool
	// Published only applies to listing datasets
	Published bool
	// ShowNumVersions only applies to listing datasets
	ShowNumVersions bool
}

// NewListParams creates a ListParams from page & pagesize, pages are 1-indexed
// (the first element is 1, not 0), NewListParams performs the conversion
func NewListParams(orderBy string, page, pageSize int) ListParams {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
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
		size = DefaultPageSize
	}
	number = lp.Offset/size + 1
	return util.NewPage(number, size)
}

// PushParams holds parameters for pushing daginfo to remotes
type PushParams struct {
	Ref           string
	RemoteName    string
	PinOnComplete bool
}

// ReceiveParams hold parameters for receiving daginfo's when running as a remote
type ReceiveParams struct {
	Peername  string
	Name      string
	ProfileID profile.ID
	DagInfo   *dag.Info
}

// ReceiveResult is the result of receiving a posted dataset when running as a remote
type ReceiveResult struct {
	Success      bool
	RejectReason string
	SessionID    string
	Diff         *dag.Manifest
}

// CompleteParams holds parameters to send when completing a dsync sent to a remote
type CompleteParams struct {
	SessionID string
}
