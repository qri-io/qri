package api

import (
	"testing"
)

func TestSearchHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	searchCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", mustFile(t, "testdata/searchRequest.json")},
		{"DELETE", "/", nil},
	}

	proh := NewSearchHandlers(node)
	runHandlerTestCases(t, "search", proh.SearchHandler, searchCases)
}
