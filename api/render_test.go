package api

import (
	"context"
	"testing"

	"github.com/qri-io/dataset"
)

func TestRenderHandler(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inst := newTestInstanceWithProfileFromNode(ctx, node)

	cases := []handlerTestCase{
		{"OPTIONS", "/render", nil},
		{"GET", "/render/me/movies?viz=true", nil},
	}

	h := NewRenderHandlers(inst)
	runHandlerTestCases(t, "render", h.RenderHandler, cases, false)
}

func TestRenderReadmeHandler(t *testing.T) {
	run := NewAPITestRunner(t)
	defer run.Delete()

	// Save a version of the dataset
	ds := run.BuildDataset("render_readme_test")
	ds.Meta = &dataset.Meta{Title: "title one"}
	ds.Readme = &dataset.Readme{ScriptBytes: []byte("# hi\n\ntest")}
	run.SaveDataset(ds, "testdata/cities/data.csv")

	// Render the dataset
	h := run.NewRenderHandlers()
	actualStatusCode, actualBody := APICall("/render/peer/render_readme_test", h.RenderHandler)
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody := `<h1>hi</h1>

<p>test</p>
`
	if expectBody != actualBody {
		t.Errorf("expected body {%s}, got {%s}", expectBody, actualBody)
	}
}
