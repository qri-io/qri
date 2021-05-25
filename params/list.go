// package params defines generic input parameter structures
package params

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
