package actions

import (
	"io/ioutil"
	"os"
	"path/filepath"
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

	tfPath := filepath.Join(os.TempDir(), "transform.star")
	defer os.RemoveAll(tfPath)
	data := `
def transform(ds,ctx):
	return [1,2,3]
`
	if err := ioutil.WriteFile(tfPath, []byte(data), 0777); err != nil {
		t.Fatal(err.Error())
	}

	ds := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: dataset.JSONDataFormat,
			Schema: dataset.BaseSchemaArray,
		},
		Transform: &dataset.Transform{
			Syntax:     "starlark",
			ScriptPath: tfPath,
		},
	}

	if _, err := ExecTransform(node, ds, nil, nil); err != nil {
		t.Error(err.Error())
	}
}
