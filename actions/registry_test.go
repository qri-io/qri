package actions

import (
	"testing"
	"time"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestRegistry(t *testing.T) {
	// to keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }
	defer func() {
		dsfs.Timestamp = prevTs
	}()

	reg := regmock.NewMemRegistry()
	regClient, regServer := regmock.NewMockServerRegistry(reg)
	defer regServer.Close()

	node := newTestNodeRegClient(t, regClient)
	ref := addCitiesDataset(t, node)

	if err := Publish(node, ref); err != nil {
		t.Error(err.Error())
	}

	cities := repo.DatasetRef{
		Peername: "me",
		Name:     "cities",
	}
	if err := RegistryDataset(node, &cities); err != nil {
		t.Error(err.Error())
	}

	expect := "/map/QmWhfkrNFGAy4Cbqo5DoC4Lipen3epjLYHZJrtsLy6hM2o"
	if expect != cities.Path {
		t.Errorf("error getting dataset from registry, expected path to be '%s', got %s", expect, cities.Path)
	}
	if cities.Dataset == nil {
		t.Errorf("error getting dataset from registry, dataset is nil")
	}
	if cities.Published != true {
		t.Errorf("error getting dataset from registry, expected published to be 'true'")
	}

	ref2 := addFlourinatedCompoundsDataset(t, node)
	if err := Publish(node, ref2); err != nil {
		t.Error(err.Error())
	}

	refs, err := RegistryList(node, 0, 0)
	if err != nil {
		t.Error(err.Error())
	}

	if len(refs) != 2 {
		t.Errorf("RegistryList should return two datasets, currently returns %d", len(refs))
	}

	if err := Unpublish(node, ref); err != nil {
		t.Error(err.Error())
	}

	refs, err = RegistryList(node, 0, 0)
	if err != nil {
		t.Error(err.Error())
	}

	if len(refs) != 1 {
		t.Errorf("RegistryList should return one dataset, currently returns %d", len(refs))
	}
}
