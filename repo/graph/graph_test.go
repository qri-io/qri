package graph

import (
	"fmt"
	"github.com/qri-io/dataset/dsgraph"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

var (
	ds1 = &dataset.Dataset{
		Title:    "dataset 1",
		Previous: datastore.NewKey(""),
	}
	ds2 = &dataset.Dataset{
		Title:    "dataset 2",
		Previous: datastore.NewKey(""),
	}
)

func TestRepoGraph(t *testing.T) {
	store := memfs.NewMapstore()
	p := &profile.Profile{}

	r, err := repo.NewMemRepo(p, store, nil, nil)
	if err != nil {
		t.Errorf("error creating test repo: %s", err.Error())
		return
	}

	data1p, _ := store.Put(memfs.NewMemfileBytes("data1", []byte("dataset_1")), true)
	fmt.Println(data1p.String())
	ds1.Data = data1p
	ds1j, _ := ds1.MarshalJSON()
	ds1p, err := store.Put(memfs.NewMemfileBytes("ds1", ds1j), true)
	if err != nil {
		t.Errorf("error putting dataset: %s", err.Error())
		return
	}
	r.PutDataset(ds1p, ds1)
	r.PutName("ds1", ds1p)

	data2p, _ := store.Put(memfs.NewMemfileBytes("data2", []byte("dataset_2")), true)
	ds2.Data = data2p
	ds2j, _ := ds2.MarshalJSON()
	ds2p, err := store.Put(memfs.NewMemfileBytes("ds2", ds2j), true)
	if err != nil {
		t.Errorf("error putting dataset: %s", err.Error())
		return
	}
	r.PutDataset(ds2p, ds2)
	r.PutName("ds2", ds2p)

	node, err := RepoGraph(r)
	if err != nil {
		t.Errorf("error generating repo graph: %s", err.Error())
		return
	}

	dsgraph.Walk(node, 0, func(n *dsgraph.Node) error {
		fmt.Println(n.Type, n.Path, n.Links)
		return nil
	})
}
