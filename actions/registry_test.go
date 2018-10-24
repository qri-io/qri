package actions

import (
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestRegistry(t *testing.T) {
	regClient, regServer := regmock.NewMockServer()
	defer regServer.Close()

	mr, err := repo.NewMemRepo(testPeerProfile, cafs.NewMapstore(), profile.NewMemStore(), regClient)
	if err != nil {
		t.Fatal(err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	tc, err := dstest.NewTestCaseFromDir(testdataPath("cities"))
	if err != nil {
		t.Error(err.Error())
		return
	}

	ref, _, err := SaveDataset(node, tc.Name, tc.Input, tc.BodyFile(), nil, false, true)
	if err != nil {
		t.Fatal(err.Error())

	}

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
