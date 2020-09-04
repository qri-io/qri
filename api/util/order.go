package util

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var (
	// OrderASC defines the ascending order keyword
	OrderASC = "asc"
	// OrderDESC defines the descending order keyword
	OrderDESC = "desc"
)

// OrderBy represents ordering information
type OrderBy []Order

// Order represents ordering information for a single key
type Order struct {
	Key       string
	Direction string
}

// NewOrder constructs a basic order struct
func NewOrder(key, orderDirection string) Order {
	if orderDirection != OrderASC && orderDirection != OrderDESC {
		orderDirection = OrderDESC
	}
	return Order{
		Key:       key,
		Direction: orderDirection,
	}
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

// String implements the stringer interface for Order
func (o Order) String() string {
	return fmt.Sprintf("%s,%s", o.Key, o.Direction)
}

// SetQueryParams adds order by info to a url as query parameters
func (o OrderBy) SetQueryParams(u *url.URL) *url.URL {
	q := u.Query()
	if len(o) > 0 {
		q.Set("orderBy", o.String())
	}

	u.RawQuery = q.Encode()
	return u
}

// OrderByFromRequest extracts orderBy params from an http request
func OrderByFromRequest(r *http.Request) OrderBy {
	orderBy := r.FormValue("orderBy")
	return NewOrderByFromString(orderBy, nil)
}

// OrderByFromRequestWithKeys extracts orderBy params from an
// http request and only takes the specified keys
func OrderByFromRequestWithKeys(r *http.Request, validKeys []string) OrderBy {
	orderBy := r.FormValue("orderBy")
	return NewOrderByFromString(orderBy, validKeys)
}

// NewOrderByFromString converts a commaa delimited string to an OrderBy struct
func NewOrderByFromString(orderBy string, validKeys []string) OrderBy {
	orderComponents := strings.Split(orderBy, ",")
	if orderBy == "" || len(orderComponents) == 0 || (len(orderComponents) != 1 && len(orderComponents)%2 == 1) {
		return []Order{}
	}
	res := []Order{}
	if len(orderComponents) == 1 {
		res = append(res, NewOrder(strings.TrimSpace(orderComponents[0]), OrderDESC))
		return res
	}
	for i := 0; i <= len(orderComponents)/2; i += 2 {
		orderDirection := ""
		if strings.ToLower(orderComponents[i+1]) == OrderASC {
			orderDirection = OrderASC
		} else {
			orderDirection = OrderDESC
		}
		orderKey := strings.TrimSpace(orderComponents[i])
		if validKeys != nil {
			for _, vkey := range validKeys {
				if vkey == orderKey {
					res = append(res, NewOrder(orderKey, orderDirection))
					break
				}
			}
		} else {
			res = append(res, NewOrder(orderKey, orderDirection))
		}
	}

	return res
}
