package api

import (
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/lib"
	reporef "github.com/qri-io/qri/repo/ref"
)

func TestRenderHandler(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	inst := newTestInstanceWithProfileFromNode(node)

	cases := []handlerTestCase{
		{"OPTIONS", "/render", nil},
		{"GET", "/render/me/movies?viz=true", nil},
	}

	h := NewRenderHandlers(inst)
	runHandlerTestCases(t, "render", h.RenderHandler, cases, false)
}

func TestRenderReadmeHandler(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	inst := newTestInstanceWithProfileFromNode(node)
	h := NewRenderHandlers(inst)
	dsm := lib.NewDatasetMethods(inst)

	// TODO(dlong): Copied from fsi_test, refactor into a common utility
	saveParams := lib.SaveParams{
		Ref: "me/render_readme_test",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "title one",
			},
			Readme: &dataset.Readme{
				ScriptBytes: []byte("# hi\n\ntest"),
			},
		},
		BodyPath: "testdata/cities/data.csv",
	}
	res := reporef.DatasetRef{}
	if err := dsm.Save(&saveParams, &res); err != nil {
		t.Fatal(err)
	}

	// Render the dataset
	actualStatusCode, actualBody := APICall(
		"/render/peer/render_readme_test",
		h.RenderHandler)
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
