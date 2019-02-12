package actions

import (
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestExecTransform(t *testing.T) {
	regClient, regServer := regmock.NewMockServer()
	defer regServer.Close()

	mr, err := repo.NewMemRepo(testPeerProfile, cafs.NewMapstore(), qfs.NewMemFS(), profile.NewMemStore(), regClient)
	if err != nil {
		t.Fatal(err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	ds := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
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
	ds.Transform.SetScriptFile(qfs.NewMemfileBytes("transform.star", data))

	if err := ExecTransform(node, ds, nil, nil); err != nil {
		t.Error(err.Error())
	}
}
