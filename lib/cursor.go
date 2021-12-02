package lib

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Cursor represents a position into a list of paginated results. Users
// of this interface can retrieve all results by repeatedly calling
// `cursor.Next()` until it returns ErrCursorComplete. When this error is
// returned, the other returned value may still have data; users should
// append this to their total collection of results.
//
// The implementation of a Cursor contains info about the method being
// called, as well as whatever input parameters that method requires.
// Moving to the next page of results is done by adjusting the input
// parameters as needed to retrieve the next page.
//
// Methods that want to create and return a Cursor are encouraged to use
// the helper `scope.MakeCursor`. The arguments to this method are the
// number of results being returned in the current page, as well as
// the input parameters to the method, after being adjusted. This method
// will correctly handle the invariant that no Cursor should be returned
// if the returned list of items is empty.
//
// Finally, a Cursor can be serialized for use in HTTP requests by using
// the ToParams method. This basically just converts the input params
// into a map of key value pairs. If passed as a POST json body, the
// receiving HTTP server will be able to convert them back into input
// params to retrieve the next page of results.

var (
	// ErrCursorComplete is returned by Cursor.Next when the cursor is completed
	ErrCursorComplete = errors.New("cursor is complete")
)

// Cursor is used to paginate results for methods that support it
type Cursor interface {
	Next(ctx context.Context) (interface{}, error)
	ToParams() (map[string]string, error)
}

type cursor struct {
	d        dispatcher
	method   string
	nextPage interface{}
}

// Next will fetch the next page of results, or return ErrCursorComplete if no
// results are left
func (c cursor) Next(ctx context.Context) (interface{}, error) {
	res, cur, err := c.d.Dispatch(ctx, c.method, c.nextPage)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, ErrCursorComplete
	}
	return res, nil
}

// ToParams returns the cursor's input params as a map of strings to strings
func (c cursor) ToParams() (map[string]string, error) {
	params := make(map[string]string)
	target := reflect.ValueOf(c.nextPage)
	if target.Kind() == reflect.Ptr {
		target = target.Elem()
	}
	for i := 0; i < target.NumField(); i++ {
		// Lowercase the key in order to make matching case-insensitive.
		fieldName := target.Type().Field(i).Name
		lowerName := strings.ToLower(fieldName)
		fieldTag := target.Type().Field(i).Tag
		if fieldTag != "" && fieldTag.Get("json") != "" {
			jsonName := fieldTag.Get("json")
			pos := strings.Index(jsonName, ",")
			if pos != -1 {
				jsonName = jsonName[:pos]
			}
			lowerName = strings.ToLower(jsonName)
		}
		v := target.Field(i)
		if v.IsZero() {
			continue
		}
		params[lowerName] = fmt.Sprintf("%v", v)
	}
	return params, nil
}
