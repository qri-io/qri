package actions

import (
	"context"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestExecTransform(t *testing.T) {
	ctx := context.Background()
	store := cafs.NewMapstore()
	mr, err := repo.NewMemRepo(testPeerProfile, store, qfs.NewMemFS(store), profile.NewMemStore())
	if err != nil {
		t.Fatal(err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	prev := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	next := &dataset.Dataset{
		Transform: &dataset.Transform{
			Syntax:  "starlark",
			Config:  map[string]interface{}{"foo": "config"},
			Secrets: map[string]string{"bar": "secret"},
		},
	}

	data := []byte(`
def transform(ds,ctx):
	ctx.get_config("foo")
	ctx.get_secret("bar")
	ds.set_body([1,2,3])
	`)
	next.Transform.SetScriptFile(qfs.NewMemfileBytes("transform.star", data))

	if err := ExecTransform(ctx, node, next, prev, nil, nil); err != nil {
		t.Error(err.Error())
	}
}
