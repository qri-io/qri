package repo

import (
	"fmt"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo/profile"
)

func TestRepoGraph(t *testing.T) {
	r, err := makeTestRepo()
	if err != nil {
		t.Errorf("error making test repo: %s", err.Error())
		return
	}
	nodes, err := RepoGraph(r)
	if err != nil {
		t.Errorf("error generating repo graph: %s", err.Error())
		return
	}

	expect := 9
	count := 0
	for range nodes {
		count++
	}
	if count != expect {
		t.Errorf("node count mismatch. expected: %d, got: %d", expect, count)
	}
}

func TestQueriesMap(t *testing.T) {
	r, err := makeTestRepo()
	if err != nil {
		t.Errorf("error making test repo: %s", err.Error())
		return
	}
	node, err := RepoGraph(r)
	if err != nil {
		t.Errorf("error generating repo graph: %s", err.Error())
		return
	}

	qs := QueriesMap(node)
	expect := 1
	if len(qs) != expect {
		t.Errorf("query count mismatch, expected: %d, got: %d", expect, len(qs))
	}
}

func TestDataNodes(t *testing.T) {
	r, err := makeTestRepo()
	if err != nil {
		t.Errorf("error making test repo: %s", err.Error())
		return
	}
	node, err := RepoGraph(r)
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
		Title:    "dataset 1",
		Previous: datastore.NewKey(""),
	}
	ds2 := &dataset.Dataset{
		Title:    "dataset 2",
		Previous: datastore.NewKey(""),
		Query: &dataset.Query{
			Abstract: &dataset.AbstractQuery{
				Syntax:    "sql",
				Statement: "select * from a,b where b.id = 'foo'",
			},
			Resources: map[string]*dataset.Dataset{
				"a": dataset.NewDatasetRef(datastore.NewKey("/path/to/a")),
				"b": dataset.NewDatasetRef(datastore.NewKey("/path/to/b")),
			},
		},
	}
	store := memfs.NewMapstore()
	p := &profile.Profile{}

	r, err := NewMemRepo(p, store, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating test repo: %s", err.Error())
	}

	data1p, _ := store.Put(memfs.NewMemfileBytes("data1", []byte("dataset_1")), true)
	ds1.Data = data1p
	ds1p, err := dsfs.SaveDataset(store, ds1, true)
	if err != nil {
		return nil, fmt.Errorf("error putting dataset: %s", err.Error())
	}
	r.PutDataset(ds1p, ds1)
	r.PutName("ds1", ds1p)

	data2p, _ := store.Put(memfs.NewMemfileBytes("data2", []byte("dataset_2")), true)
	ds2.Data = data2p
	ds2p, err := dsfs.SaveDataset(store, ds2, true)
	if err != nil {
		return nil, fmt.Errorf("error putting dataset: %s", err.Error())
	}
	r.PutDataset(ds2p, ds2)
	r.PutName("ds2", ds2p)

	return r, nil
}
