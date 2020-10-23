package base

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
)

// Select loads a dataset value specified by case.Sensitve.dot.separated.paths
func Select(ctx context.Context, fs qfs.Filesystem, ref dsref.Ref, valuePath string) (interface{}, error) {
	ds, err := dsfs.LoadDataset(ctx, fs, ref.Path)
	if err != nil {
		return nil, err
	}

	if valuePath == "" {
		return ds, nil
	}

	v, err := pathValue(ds, valuePath)
	if err != nil {
		return nil, err
	}
	return v.Interface(), nil
}

// ApplyPath gets a dataset value by applying a case.Sensitve.dot.separated.path
// ApplyPath cannot select file fields
func ApplyPath(ds *dataset.Dataset, path string) (interface{}, error) {
	var value reflect.Value
	value, err := pathValue(ds, path)
	if err != nil {
		return nil, err
	}
	return value.Interface(), nil
}

func pathValue(ds *dataset.Dataset, path string) (elem reflect.Value, err error) {
	elem = reflect.ValueOf(ds)

	for _, sel := range strings.Split(path, ".") {
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		switch elem.Kind() {
		case reflect.Struct:
			elem = elem.FieldByNameFunc(func(str string) bool {
				return strings.ToLower(str) == sel
			})
		case reflect.Slice:
			index, err := strconv.Atoi(sel)
			if err != nil {
				return elem, fmt.Errorf("invalid index value: %s", sel)
			}
			elem = elem.Index(index)
		case reflect.Map:
			for _, key := range elem.MapKeys() {
				// we only support strings as keys
				if strings.ToLower(key.String()) == sel {
					return elem.MapIndex(key), nil
				}
			}
			return elem, fmt.Errorf("invalid selection path: %s", path)
		}

		if elem.Kind() == reflect.Invalid {
			return elem, fmt.Errorf("invalid selection path: %s", path)
		}
	}

	return elem, nil
}
