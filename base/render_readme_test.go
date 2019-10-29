package base

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qfs"
)

func TestRenderReadme(t *testing.T) {
	ctx := context.Background()

	f := qfs.NewMemfileBytes("test.md", []byte(`# hi

three things:

* one
* two
* three`))
	htmlStr, err := RenderReadme(ctx, f)
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
	if diff := cmp.Diff(expectStr, htmlStr); diff != "" {
		t.Errorf("body component (-want +got):\n%s", diff)
	}
}

func TestRenderReadmeWithScriptTag(t *testing.T) {
	ctx := context.Background()

	f := qfs.NewMemfileBytes("test.md", []byte(`# a script

<script>alert('hi');</script>

done`))
	htmlStr, err := RenderReadme(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	expectStr := `<h1>a script</h1>



<p>done</p>
`
	if diff := cmp.Diff(expectStr, htmlStr); diff != "" {
		t.Errorf("body component (-want +got):\n%s", diff)
	}
}
