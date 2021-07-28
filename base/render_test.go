package base

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qfs"
)

func TestRender(t *testing.T) {
	t.Skip("TODO (b5) - need to fix qfs / repo connection for this to work")
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	ds, err := NewTestDatasetLoader(r.Filesystem(), r).LoadDataset(ctx, ref.String())
	if err != nil {
		t.Fatal(err)
	}

	_, err = Render(ctx, r, ds, nil)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestRenderReadme(t *testing.T) {
	ctx := context.Background()

	f := qfs.NewMemfileBytes("test.md", []byte(`# hi

three things:

* one
* two
* three`))
	htmlBytes, err := RenderReadme(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	expectStr := `<h1>hi</h1>

<p>three things:</p>

<ul>
<li>one</li>
<li>two</li>
<li>three</li>
</ul>
`
	if diff := cmp.Diff(expectStr, string(htmlBytes)); diff != "" {
		t.Errorf("body component (-want +got):\n%s", diff)
	}
}

func TestRenderReadmeWithScriptTag(t *testing.T) {
	ctx := context.Background()

	f := qfs.NewMemfileBytes("test.md", []byte(`# a script

<script>alert('hi');</script>

done`))
	htmlBytes, err := RenderReadme(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	expectStr := `<h1>a script</h1>



<p>done</p>
`
	if diff := cmp.Diff(expectStr, string(htmlBytes)); diff != "" {
		t.Errorf("body component (-want +got):\n%s", diff)
	}
}
