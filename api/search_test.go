package api

import (
	"testing"
)

func TestSearchHandlers(t *testing.T) {
	r, teardown := newTestRepo(t)
	defer teardown()

	searchCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", mustFile(t, "testdata/searchRequest.json")},
		{"DELETE", "/", nil},
	}

	proh := NewSearchHandlers(r)
	runHandlerTestCases(t, "search", proh.SearchHandler, searchCases)
}
