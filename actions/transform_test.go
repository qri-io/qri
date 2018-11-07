package actions

import (
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestExecTransform(t *testing.T) {
	regClient, regServer := regmock.NewMockServer()
	defer regServer.Close()

	mr, err := repo.NewMemRepo(testPeerProfile, cafs.NewMapstore(), profile.NewMemStore(), regClient)
	if err != nil {
		t.Fatal(err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	data := []byte(`
def transform(ds,ctx):
	return [1,2,3]
`)
	script := cafs.NewMemfileBytes("transform.star", data)

	ds := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: dataset.JSONDataFormat,
			Schema: dataset.BaseSchemaArray,
		},
		Transform: &dataset.Transform{
			Syntax: "starlark",
		},
	}

	if _, err := ExecTransform(node, ds, script, nil, nil, nil); err != nil {
		t.Error(err.Error())
	}
}
