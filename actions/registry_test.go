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
	if err := Unpublish(node, ref); err != nil {
		t.Error(err.Error())
	}
}
