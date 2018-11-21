package lib

import (
	"testing"

	"github.com/qri-io/registry/regserver/mock"
)

func TestRegistryRequests(t *testing.T) {
	reg := mock.NewMemRegistry()
	cli, _ := mock.NewMockServerRegistry(reg)
	node := newTestQriNodeRegClient(t, cli)

	ref := addCitiesDataset(t, node)
	ref2 := addNowTransformDataset(t, node)

	req := NewRegistryRequests(node, nil)

	params := &PublishParams{Ref: ref, Pin: true}
	done := false
	if err := req.Publish(params, &done); err != nil {
		t.Fatal(err)
	}

	params = &PublishParams{Ref: ref2, Pin: true}
	done = false
	if err := req.Publish(params, &done); err != nil {
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
