package actions

import (
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/fs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestExecTransform(t *testing.T) {
	regClient, regServer := regmock.NewMockServer()
	defer regServer.Close()

	mr, err := repo.NewMemRepo(testPeerProfile, cafs.NewMapstore(), fs.NewMemFS(), profile.NewMemStore(), regClient)
	if err != nil {
		t.Fatal(err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	data := []byte(`
def transform(ds,ctx):
	ctx.get_config("foo")
	ctx.get_secret("bar")
	ds.set_body([1,2,3])
`)
	script := fs.NewMemfileBytes("transform.star", data)

	ds := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
		Transform: &dataset.Transform{
			Syntax: "starlark",
		},
	}

	if _, err := ExecTransform(node, ds, script, nil, map[string]string{"foo": "config"}, map[string]interface{}{"bar": "secret"}, nil, nil); err != nil {
		t.Error(err.Error())
	}
}
