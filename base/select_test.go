package base

import (
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestSelect(t *testing.T) {
	regClient, regServer := regmock.NewMockServer()
	defer regServer.Close()

	mr, err := repo.NewMemRepo(testPeerProfile, cafs.NewMapstore(), profile.NewMemStore(), regClient)
	if err != nil {
		t.Fatal(err.Error())
	}

	tc, err := dstest.NewTestCaseFromDir(testdataPath("cities"))
	if err != nil {
		t.Error(err.Error())
		return
	}

	ref, err := CreateDataset(mr, tc.Name, tc.Input, tc.BodyFile(), false)
	if err != nil {
		t.Fatal(err.Error())
	}

	if _, err := Select(mr, ref, "commit"); err != nil {
		t.Error(err.Error())
	}
}
