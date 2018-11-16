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

	req := NewRegistryRequests(node, nil)

	params := &PublishParams{Ref: ref, Pin: true}
	done := false
	if err := req.Publish(params, &done); err != nil {
		t.Fatal(err)
	}

	if reg.Datasets.Len() != 1 {
		t.Errorf("expected registry to have 1 dataset. got: %d", reg.Datasets.Len())
	}

	if err := req.Unpublish(&ref, &done); err != nil {
		t.Fatal(err)
	}

	if reg.Datasets.Len() != 0 {
		t.Errorf("expected registry to have 0 datasets. got: %d", reg.Datasets.Len())
	}

}
