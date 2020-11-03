package api

import (
	"context"
	"testing"
)

func TestSQLHandler(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inst := newTestInstanceWithProfileFromNode(ctx, node)
	h := NewSQLHandlers(inst, false)

	cases := []handlerTestCase{
		{"GET", "/sql?query=select%20*%20from%20me/movies%20m%20order%20by%20m.title%20limit%201", nil},
	}
	runHandlerTestCases(t, "sql", h.QueryHandler("/sql"), cases, false)

	jsonCases := []handlerTestCase{
		{"POST", "/sql", []byte(`{}`)},
		{"POST", "/sql", []byte(`{"query":"select * from me/movies m order by m.title limit 1"}`)},
	}
	runHandlerTestCases(t, "sql", h.QueryHandler("/sql"), jsonCases, true)
}
