package api

import (
	"context"
	"testing"

	"github.com/qri-io/qri/lib"
	regmock "github.com/qri-io/qri/registry/regserver"
)

func TestSearchHandlers(t *testing.T) {
	t.Skip("TODO (b5): restore. need more work in lib.NewInstance constructor")
	node, teardown := newTestNode(t)
	defer teardown()

	rc, _ := regmock.NewMockServer()

	inst, err := lib.NewInstance(context.Background(), "./repo", lib.OptQriNode(node), lib.OptRegistryClient(rc))
	if err != nil {
		t.Fatal(err)
	}

	searchCases := []handlerTestCase{
		// TODO (b5): lol wut Get requests don't have bodies
		{"GET", "/", mustFile(t, "testdata/searchRequest.json")},
		{"DELETE", "/", nil},
	}

	proh := NewSearchHandlers(inst)
	runHandlerTestCases(t, "search", proh.SearchHandler, searchCases, true)
}
