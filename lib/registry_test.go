package lib

import (
	"testing"
	"time"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/registry/regserver/mock"
)

func TestRegistryRequests(t *testing.T) {
	// to keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }
	defer func() {
		dsfs.Timestamp = prevTs
	}()

	reg := mock.NewMemRegistry()
	cli, _ := mock.NewMockServerRegistry(reg)
	node := newTestQriNodeRegClient(t, cli)

	ref := addCitiesDataset(t, node)
	ref2 := addNowTransformDataset(t, node)

	req := NewRegistryRequests(node, nil)

	done := false
	if err := req.Publish(&ref, &done); err != nil {
		t.Fatal(err)
	}

	// test getting a dataset from the registry
	citiesRef := repo.DatasetRef{
		Peername: "me",
		Name:     "cities",
	}
	citiesRes := repo.DatasetRef{}
	if err := req.GetDataset(&citiesRef, &citiesRes); err != nil {
		t.Error(err)
	}

	expect := "/map/QmWhfkrNFGAy4Cbqo5DoC4Lipen3epjLYHZJrtsLy6hM2o"
	if expect != citiesRes.Path {
		t.Errorf("error getting dataset from registry, expected path to be '%s', got %s", expect, citiesRes.Path)
	}
	if citiesRes.Dataset == nil {
		t.Errorf("error getting dataset from registry, dataset is nil")
	}
	if citiesRes.Published != true {
		t.Errorf("error getting dataset from registry, expected published to be 'true'")
	}

	done = false
	if err := req.Publish(&ref2, &done); err != nil {
		t.Fatal(err)
	}

	rlp := &RegistryListParams{}
	if err := req.List(rlp, &done); err != nil {
		t.Error(err)
	}
	if len(rlp.Refs) != 2 {
		t.Errorf("expected registry to have 2 datasets. got: %d", reg.Datasets.Len())
	}

	if err := req.Unpublish(&ref, &done); err != nil {
		t.Fatal(err)
	}

	rlp = &RegistryListParams{}

	if err := req.List(rlp, &done); err != nil {
		t.Error(err)
	}
	if len(rlp.Refs) != 1 {
		t.Errorf("expected registry to have 1 dataset. got: %d", reg.Datasets.Len())
	}

}
