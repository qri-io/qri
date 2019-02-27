package api

import (
	"testing"
)

func TestSearchHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	searchCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		// TODO (b5): lol wut Get requests don't have bodies
		{"GET", "/", mustFile(t, "testdata/searchRequest.json")},
		{"DELETE", "/", nil},
	}

	proh := NewSearchHandlers(node)
	runHandlerTestCases(t, "search", proh.SearchHandler, searchCases, true)
}
