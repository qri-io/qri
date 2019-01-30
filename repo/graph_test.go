package repo

import (
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/fs"
	"github.com/qri-io/qri/repo/profile"
)

func TestGraph(t *testing.T) {
	r, err := makeTestRepo()
	if err != nil {
		t.Errorf("error making test repo: %s", err.Error())
		return
	}
	nodes, err := Graph(r)
	if err != nil {
		t.Errorf("error generating repo graph: %s", err.Error())
		return
	}

	expect := 8
	count := 0
	for range nodes {
		count++
	}
	if count != expect {
		t.Errorf("node count mismatch. expected: %d, got: %d", expect, count)
	}
}

func TestDataNodes(t *testing.T) {
	r, err := makeTestRepo()
	if err != nil {
		t.Errorf("error making test repo: %s", err.Error())
		return
	}
	node, err := Graph(r)
	if err != nil {
		t.Errorf("error generating repo graph: %s", err.Error())
		return
	}

	dn := DataNodes(node)
	expect := 2
	if len(dn) != expect {
		t.Errorf("data node mismatch, expected: %d, got: %d", expect, len(dn))
	}
}

func makeTestRepo() (Repo, error) {
	ds1 := &dataset.Dataset{
		Meta: &dataset.Meta{
			Title: "dataset 1",
		},
		Commit: &dataset.Commit{
			Message: "foo",
		},
		PreviousPath: "",
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaObject,
		},
	}
	ds2 := &dataset.Dataset{
		Meta: &dataset.Meta{
			Title: "dataset 2",
		},
		Commit: &dataset.Commit{
			Message: "bar",
		},
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaObject,
		},
		Transform: &dataset.Transform{
			Syntax:     "sql",
			ScriptPath: "foo.sky",
			Resources: map[string]*dataset.TransformResource{
				"a": &dataset.TransformResource{Path: "/path/to/a"},
				"b": &dataset.TransformResource{Path: "/path/to/b"},
			},
		},
		PreviousPath: "",
	}
	store := cafs.NewMapstore()
	p := &profile.Profile{}

	r, err := NewMemRepo(p, store, fs.NewMemFS(), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating test repo: %s", err.Error())
	}

	privKey, err := crypto.UnmarshalPrivateKey(testPk)
	if err != nil {
		err = fmt.Errorf("error unmarshaling private key: %s", err.Error())
		return nil, err
	}

	pro := &profile.Profile{
		ID:       profile.IDB58MustDecode("QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"),
		Peername: "peer",
		PrivKey:  privKey,
	}
	r.SetProfile(pro)

	data1f := fs.NewMemfileBytes("data1", []byte("dataset_1"))

	ds1p, err := dsfs.WriteDataset(store, ds1, data1f, true)
	if err != nil {
		return nil, fmt.Errorf("error putting dataset: %s", err.Error())
	}
	if err := r.PutRef(DatasetRef{ProfileID: pro.ID, Peername: pro.Peername, Name: "ds1", Path: ds1p}); err != nil {
		return nil, err
	}

	data2f := fs.NewMemfileBytes("data2", []byte("dataset_2"))
	ds2p, err := dsfs.WriteDataset(store, ds2, data2f, true)
	if err != nil {
		return nil, fmt.Errorf("error putting dataset: %s", err.Error())
	}
	if err := r.PutRef(DatasetRef{ProfileID: pro.ID, Peername: pro.Peername, Name: "ds2", Path: ds2p}); err != nil {
		return nil, err
	}

	return r, nil
}
