package lib

import (
	"reflect"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/profile"
)

// DefaultLimit is the max number of items in a response if no
// Limit param is provided to a paginated method
const DefaultLimit = 100

// EmptyParams is for methods that don't need any input
type EmptyParams struct {
}

// NZDefaultSetter modifies zero values to non-zero defaults when called
type NZDefaultSetter interface {
	SetNonZeroDefaults()
}

// ListParams is the general input for any sort of Paginated Request
// ListParams define limits & offsets, not pages & page sizes.
// TODO - rename this to PageParams.
type ListParams struct {
	// TODO(b5): what is this being used for?
	ProfileID profile.ID `json:"-" docs:"hidden"`
	// term to filter list by; e.g. "population"
	Term string `json:"term,omitempty"`
	// username to filter collection by; e.g. "ramfox"
	Username string `json:"username,omitempty"`
	// field name to order list by; e.g. "created"
	OrderBy string `json:"orderBy,omitempty"`
	// maximum number of datasets to use. use -1 to list all datasets; e.g. 50
	Limit int `json:"limit"`
	// number of items to skip; e.g. 0
	Offset int `json:"offset"`
	// Public only applies to listing datasets, shows only datasets that are
	// set to visible
	Public bool `json:"public,omitempty"`
}

// SetNonZeroDefaults sets OrderBy to "created" if it's value is the empty string
func (p *ListParams) SetNonZeroDefaults() {
	if p.OrderBy == "" {
		p.OrderBy = "created"
	}
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
