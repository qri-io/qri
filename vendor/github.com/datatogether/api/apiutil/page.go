package apiutil

import "net/http"

const DEFAULT_PAGE_SIZE = 100

// a page represents pagination data & common pagination
type Page struct {
	Number int `json:"page"`
	Size   int `json:"pageSize"`
}

func NewPage(number, size int) Page {
	return Page{number, size}
}

func (p Page) Limit() int {
	return p.Size
}

func (p Page) Offset() int {
	return (p.Number - 1) * p.Size
}

// pull pagination params from an http request
func PageFromRequest(r *http.Request) Page {
	var number, size int
	if i, err := ReqParamInt("page", r); err == nil {
		number = i
	}
	if number == 0 {
		number = 1
	}

	if i, err := ReqParamInt("pageSize", r); err == nil {
		size = i
	}
	if size == 0 {
		size = DEFAULT_PAGE_SIZE
	}

	return NewPage(number, size)
}
