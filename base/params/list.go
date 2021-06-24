// Package params defines generic input parameter structures
package params

import "fmt"

// ListAll uses a limit of -1 & offset of 0 as a sentinel value for listing
// all items
var ListAll = List{Limit: -1}

// List defines input parameters for listing operations
type List struct {
	Filter  []string
	OrderBy []string
	Limit   int
	Offset  int
}

func (lp List) All() bool {
	return lp.Limit == -1 && lp.Offset == 0
}

func (lp List) Validate() error {
	if lp.Limit < 0 && lp.Limit != -1 {
		return fmt.Errorf("limit of %d is out of bounds", lp.Limit)
	}
	if lp.Offset < 0 {
		return fmt.Errorf("offset of %d is out of bounds", lp.Offset)
	}
	return nil
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

// WithFilters returns a new List struct that replaces the Filters field
func (lp List) WithFilters(f ...string) List {
	return List{
		Filter:  f,
		OrderBy: lp.OrderBy,
		Offset:  lp.Offset,
		Limit:   lp.Limit,
	}
}

// WithOrderBy returns a new List struct that replaces the OrderBy field
func (lp List) WithOrderBy(ob ...string) List {
	return List{
		Filter:  lp.Filter,
		OrderBy: ob,
		Offset:  lp.Offset,
		Limit:   lp.Limit,
	}
}
