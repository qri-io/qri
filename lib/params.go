package lib

import (
	"reflect"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
)

// EmptyParams is for methods that don't need any input
type EmptyParams struct {
}

// NZDefaultSetter modifies zero values to non-zero defaults when called
type NZDefaultSetter interface {
	SetNonZeroDefaults()
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
