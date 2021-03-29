package lib

import (
	"net/http"
	"reflect"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/profile"
)

// DefaultPageSize is the max number of items in a page if no
// Limit param is provided to a paginated method
const DefaultPageSize = 100

// NZDefaultSetter modifies zero values to non-zero defaults when called
type NZDefaultSetter interface {
	SetNonZeroDefaults()
}

// RequestUnmarshaller is an interface for deserializing from an HTTP request
type RequestUnmarshaller interface {
	UnmarshalFromRequest(r *http.Request) error
}

// ListParams is the general input for any sort of Paginated Request
// ListParams define limits & offsets, not pages & page sizes.
// TODO - rename this to PageParams.
type ListParams struct {
	ProfileID profile.ID `json:"-"`
	Term      string
	Peername  string
	OrderBy   string
	Limit     int
	Offset    int
	// RPC is a horrible hack while we work to replace the net/rpc package
	// TODO - remove this
	RPC bool
	// Public only applies to listing datasets, shows only datasets that are
	// set to visible
	Public bool
	// ShowNumVersions only applies to listing datasets
	ShowNumVersions bool
	// EnsureFSIExists controls whether to ensure references in the repo have correct FSIPaths
	EnsureFSIExists bool
	// UseDscache controls whether to build a dscache to use to list the references
	UseDscache bool

	// Raw indicates whether to return a raw string representation of a dataset list
	Raw bool
}

// SetNonZeroDefaults sets OrderBy to "created" if it's value is the empty string
func (p *ListParams) SetNonZeroDefaults() {
	if p.OrderBy == "" {
		p.OrderBy = "created"
	}
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *ListParams) UnmarshalFromRequest(r *http.Request) error {
	lp := ListParamsFromRequest(r)
	if p == nil {
		p = &ListParams{}
	}
	if !p.Raw {
		lp.Raw = r.FormValue("raw") == "true"
	} else {
		lp.Raw = p.Raw
	}
	if p.Peername == "" {
		lp.Peername = r.FormValue("peername")
	} else {
		lp.Peername = p.Peername
	}
	if p.Term == "" {
		lp.Term = r.FormValue("term")
	} else {
		lp.Term = p.Term
	}
	*p = lp
	return nil
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
	if i := util.ReqParamInt(r, "page", 0); i != 0 {
		page = i
	}
	if i := util.ReqParamInt(r, "pageSize", 0); i != 0 {
		pageSize = i
	}
	return NewListParams(r.FormValue("orderBy"), page, pageSize)
}

// Page converts a ListParams struct to a util.Page struct
func (p ListParams) Page() util.Page {
	var number, size int
	size = p.Limit
	if size <= 0 {
		size = DefaultPageSize
	}
	number = p.Offset/size + 1
	return util.NewPage(number, size)
}

// normalizeInputParams will look at each field of the params, and modify filepaths so that
// they are absolute paths, making them safe to send across RPC to another process
func normalizeInputParams(param interface{}) interface{} {
	typ := reflect.TypeOf(param)
	val := reflect.ValueOf(param)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}
	if typ.Kind() == reflect.Struct {
		num := typ.NumField()
		for i := 0; i < num; i++ {
			tfield := typ.Field(i)
			vfield := val.Field(i)
			qriTag := tfield.Tag.Get("qri")
			if isQriStructTag(qriTag) {
				normalizeFileField(vfield, qriTag)
			} else if qriTag != "" {
				log.Errorf("unknown qri struct tag %q", qriTag)
			}
		}
	}
	return param
}

// qri struct tags augment how fields are marshalled for dispatched methods
const (
	// QriStTagFspath means the field is a filesystem path and needs to be absolute
	QriStTagFspath = "fspath"
	// QriStTagRefOrPath means the field is either a dataset ref, or is a filesys path
	QriStTagRefOrPath = "dsrefOrFspath"
)

func isQriStructTag(text string) bool {
	return text == QriStTagFspath || text == QriStTagRefOrPath
}

func normalizeFileField(vfield reflect.Value, qriTag string) {
	interf := vfield.Interface()
	if str, ok := interf.(string); ok {
		if qriTag == QriStTagRefOrPath && dsref.IsRefString(str) {
			return
		}
		if err := qfs.AbsPath(&str); err == nil {
			vfield.SetString(str)
		}
	}
	if strList, ok := interf.([]string); ok {
		build := make([]string, 0, len(strList))
		for _, str := range strList {
			if qriTag != QriStTagRefOrPath || !dsref.IsRefString(str) {
				_ = qfs.AbsPath(&str)
			}
			build = append(build, str)
		}
		vfield.Set(reflect.ValueOf(build))
	}
}
