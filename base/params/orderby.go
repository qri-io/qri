package params

import (
	"fmt"
	"net/http"
	"strings"
)

var (
	// OrderASC defines the ascending order keyword
	OrderASC = "+"
	// OrderDESC defines the descending order keyword
	OrderDESC = "-"
)

// OrderBy represents ordering information
type OrderBy []*Order

// OrderByFromRequest extracts orderBy params from an http request
func OrderByFromRequest(r *http.Request) OrderBy {
	orderBy := r.FormValue("orderBy")
	return NewOrderByFromString(orderBy)
}

// NewOrderByFromString converts a comma delimited string to an OrderBy struct
// eg: `+name,-updated,username`
// OrderBy {
//	{ Key: "name", Direction: OrderASC },
//  { Key: "updated", Direction: OrderDESC },
//  { Key: "username", Direction: OrderASC },
// }
func NewOrderByFromString(orderBy string) OrderBy {
	orders := strings.Split(orderBy, ",")
	res := []*Order{}
	for _, orderStr := range orders {
		order := NewOrderFromString(orderStr)
		if order != nil {
			res = append(res, order)
		}
	}
	return res
}

// String implements the stringer interface for OrderBy
func (o OrderBy) String() string {
	var b strings.Builder
	for i, oi := range o {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(oi.String())
	}
	return b.String()
}

// Order represents ordering information for a single key
// Default direction is OrderASC
type Order struct {
	Key       string
	Direction string
}

// NewOrder constructs a basic order struct
func NewOrder(key, orderDirection string) *Order {
	if key == "" {
		return nil
	}
	if orderDirection != OrderASC && orderDirection != OrderDESC {
		orderDirection = OrderASC
	}
	return &Order{
		Key:       key,
		Direction: orderDirection,
	}
}

// NewOrderFromString constructs an order using a string
// Default direction is ascending
func NewOrderFromString(orderStr string) *Order {
	if orderStr == "" {
		return nil
	}
	key := orderStr
	orderDirection := ""
	if orderStr[0:1] == OrderDESC || orderStr[0:1] == OrderASC {
		orderDirection = orderStr[0:1]
		key = orderStr[1:]
	}
	return NewOrder(key, orderDirection)
}

// String implements the stringer interface for Order
func (o *Order) String() string {
	return fmt.Sprintf("%s%s", o.Direction, o.Key)
}
