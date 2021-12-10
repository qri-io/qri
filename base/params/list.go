// Package params defines generic input parameter structures
package params

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var (
	// ErrListParamsNotEmpty indicates that there are list values in the params
	ErrListParamsNotEmpty = fmt.Errorf("list params not empty")
)

type ListParams interface {
	// ListParamsFromRequest pulls list params from the URL
	ListParamsFromRequest(r *http.Request) (List, error)
}

// ListAll uses a limit of -1 & offset of 0 as a sentinel value for listing
// all items
var ListAll = List{Limit: -1}

// List defines input parameters for listing operations
type List struct {
	Filter  []string
	OrderBy OrderBy
	Limit   int
	Offset  int
}

// ListParamsFromRequest satisfies the ListParams interface
func (lp List) ListParamsFromRequest(r *http.Request) (List, error) {
	l := List{}
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		limit, err := strconv.ParseInt(limitStr, 10, 0)
		if err != nil {
			return lp, err
		}
		l.Limit = int(limit)
	}
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr != "" {
		offset, err := strconv.ParseInt(offsetStr, 10, 0)
		if err != nil {
			return lp, err
		}
		l.Offset = int(offset)
	}

	filterStr := r.URL.Query().Get("filter")
	if filterStr != "" {
		l = l.WithFilters(strings.Split(r.FormValue("filter"), ",")...)
	}

	l = l.WithOrderBy(r.URL.Query().Get("orderby"))

	if l.IsEmpty() {
		return lp, nil
	}
	if !lp.IsEmpty() {
		return lp, ErrListParamsNotEmpty
	}

	return l, nil
}

// All returns true if List is attempting to list all available items
func (lp List) All() bool {
	return lp.Limit == -1 && lp.Offset == 0
}

// Validate returns an error if any fields or field combinations are invalid
func (lp List) Validate() error {
	if lp.Limit < 0 && lp.Limit != -1 {
		return fmt.Errorf("limit of %d is out of bounds", lp.Limit)
	}
	if lp.Offset < 0 {
		return fmt.Errorf("offset of %d is out of bounds", lp.Offset)
	}
	return nil
}

// IsEmpty determines if the List struct is empty
func (lp List) IsEmpty() bool {
	return lp.Limit == 0 && lp.Offset == 0 &&
		(lp.Filter == nil || len(lp.Filter) == 0) &&
		(lp.OrderBy == nil || len(lp.OrderBy) == 0)
}

// WithOffsetLimit returns a new List struct that replaces the offset & limit
// fields
func (lp List) WithOffsetLimit(offset, limit int) List {
	return List{
		Filter:  lp.Filter,
		OrderBy: lp.OrderBy,
		Offset:  offset,
		Limit:   limit,
	}
}

// WithOrderBy returns a new List struct that replaces the OrderBy field
func (lp List) WithOrderBy(ob string) List {
	return List{
		Filter:  lp.Filter,
		OrderBy: NewOrderByFromString(ob),
		Offset:  lp.Offset,
		Limit:   lp.Limit,
	}
}

// WithFilters returns a new List struct that replaces the Filters field
func (lp List) WithFilters(f ...string) List {
	return List{
		Filter:  f,
		OrderBy: lp.OrderBy,
		Offset:  lp.Offset,
		Limit:   lp.Limit,
	}
}
