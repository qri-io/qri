package repo

import (
	"fmt"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
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
	node, err := Graph(r)
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

func TestDatasetQueries(t *testing.T) {
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

	qs := DatasetQueries(node)
	expect := 2
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
		PreviousPath: "/",
	}
	ds2 := &dataset.Dataset{
		Meta: &dataset.Meta{
			Title: "dataset 2",
		},
		Transform: &dataset.Transform{
			Syntax: "sql",
			Data:   "select * from a,b where b.id = 'foo'",
			Resources: map[string]*dataset.Dataset{
				"a": dataset.NewDatasetRef(datastore.NewKey("/path/to/a")),
				"b": dataset.NewDatasetRef(datastore.NewKey("/path/to/b")),
			},
		},
		PreviousPath: "/",
	}
	store := cafs.NewMapstore()
	p := &profile.Profile{}

	r, err := NewMemRepo(p, store, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating test repo: %s", err.Error())
	}

	data1f := cafs.NewMemfileBytes("data1", []byte("dataset_1"))
	ds1p, err := dsfs.WriteDataset(store, ds1, data1f, true)
	if err != nil {
		return nil, fmt.Errorf("error putting dataset: %s", err.Error())
	}
	r.PutDataset(ds1p, ds1)
	r.PutRef(DatasetRef{Peername: "peer", Name: "ds1", Path: ds1p.String()})

	data2f := cafs.NewMemfileBytes("data2", []byte("dataset_2"))
	ds2p, err := dsfs.WriteDataset(store, ds2, data2f, true)
	if err != nil {
		return nil, fmt.Errorf("error putting dataset: %s", err.Error())
	}
	r.PutDataset(ds2p, ds2)
	r.PutRef(DatasetRef{Peername: "peer", Name: "ds2", Path: ds2p.String()})

	return r, nil
}
