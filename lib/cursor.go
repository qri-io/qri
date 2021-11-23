package lib

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

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
