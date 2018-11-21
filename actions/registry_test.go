package actions

import (
	"testing"

	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestRegistry(t *testing.T) {
	reg := regmock.NewMemRegistry()
	regClient, regServer := regmock.NewMockServerRegistry(reg)
	defer regServer.Close()

	node := newTestNodeRegClient(t, regClient)
	ref := addCitiesDataset(t, node)

	if err := Publish(node, ref); err != nil {
		t.Error(err.Error())
	}
	if err := Status(node, ref); err != nil {
		t.Error(err.Error())
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
