package util

import (
	"net/http"
	"net/url"
	"strconv"
)

var (
	// DefaultPageSize is the number NewPage
	DefaultPageSize = 50
	// DefaultMaxPageSize is the max size a page can be by default, set to -1 to
	// ignore maximum sizes
	DefaultMaxPageSize = 100
)

// Page represents pagination information
type Page struct {
	Number      int    `json:"page,omitempty"`
	Size        int    `json:"pageSize,omitempty"`
	ResultCount int    `json:"resultCount,omitempty"`
	NextURL     string `json:"nextUrl"`
	PrevURL     string `json:"prevUrl"`
}

// NewPage constructs a basic page struct, setting sensible defaults
func NewPage(number, size int) Page {
	if number <= 0 {
		number = 1
	}

	if DefaultMaxPageSize != -1 && size > DefaultMaxPageSize {
		size = DefaultMaxPageSize
	}

	return Page{
		Number: number,
		Size:   size,
	}
}

// Limit is a convenience accessor for page size
func (p Page) Limit() int {
	return p.Size
}

// Offset calculates the starting index for pagination based on page
// size & number
func (p Page) Offset() int {
	return (p.Number - 1) * p.Size
}

// Next returns a page with the number advanced by one
func (p Page) Next() Page {
	return Page{
		Number:      p.Number + 1,
		Size:        p.Size,
		ResultCount: p.ResultCount,
		NextURL:     p.NextURL,
		PrevURL:     p.PrevURL,
	}
}

// Prev returns a page with the number decremented by 1
func (p Page) Prev() Page {
	return Page{
		Number:      p.Number - 1,
		Size:        p.Size,
		ResultCount: p.ResultCount,
		NextURL:     p.NextURL,
		PrevURL:     p.PrevURL,
	}
}

// SetQueryParams adds pagination info to a url as query parameters
func (p Page) SetQueryParams(u *url.URL) *url.URL {
	q := u.Query()
	q.Set("page", strconv.Itoa(p.Number))
	if p.Size != DefaultPageSize {
		q.Set("pageSize", strconv.Itoa(p.Size))
	}

	u.RawQuery = q.Encode()
	return u
}

// NextPageExists returns false if the next page.ResultCount is a postive
// number and the starting offset of the next page exceeds page.ResultCount
func (p Page) NextPageExists() bool {
	return p.ResultCount <= 0 || !(p.Number*p.Size >= p.ResultCount)
}

// PrevPageExists returns false if the page number is 1
func (p Page) PrevPageExists() bool {
	return p.Number > 1
}

// PageFromRequest extracts pagination params from an http request
func PageFromRequest(r *http.Request) Page {
	number := ReqParamInt(r, "page", 1)
	if number <= 0 {
		number = 1
	}

	size := ReqParamInt(r, "pageSize", DefaultPageSize)
	if size <= 0 {
		size = DefaultPageSize
	}

	if DefaultMaxPageSize != -1 && size > DefaultMaxPageSize {
		size = DefaultMaxPageSize
	}

	return Page{
		Number: number,
		Size:   size,
	}
}

// NewPageFromOffsetAndLimit converts a offset and Limit to a Page struct
func NewPageFromOffsetAndLimit(offset, limit int) Page {
	var number, size int
	size = limit
	if size <= 0 {
		size = DefaultPageSize
	}
	number = offset/size + 1
	return Page{
		Number: number,
		Size:   size,
	}
}
