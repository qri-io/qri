package actions

import (
	"context"
	"testing"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestUpdateRemoteDataset(t *testing.T) {
	// TODO (b5) - restore
}

func TestDataset(t *testing.T) {
	rmf := func(t *testing.T) repo.Repo {
		store := cafs.NewMapstore()
		testPeerProfile.PrivKey = privKey
		mr, err := repo.NewMemRepo(testPeerProfile, store, qfs.NewMemFS(), profile.NewMemStore())
		if err != nil {
			panic(err)
		}
		return mr
	}
	DatasetTests(t, rmf)
}

type RepoMakerFunc func(t *testing.T) repo.Repo
type RepoTestFunc func(t *testing.T, rmf RepoMakerFunc)

func DatasetTests(t *testing.T, rmf RepoMakerFunc) {
	for _, test := range []RepoTestFunc{
		testSaveDataset,
		testReadDataset,
		testRenameDataset,
	} {
		test(t, rmf)
	}
}

func testSaveDataset(t *testing.T, rmf RepoMakerFunc) {
	createDataset(t, rmf)
}

func createDataset(t *testing.T, rmf RepoMakerFunc) (*p2p.QriNode, repo.DatasetRef) {
	r := rmf(t)
	r.SetProfile(testPeerProfile)
	n, err := p2p.NewQriNode(r, config.DefaultP2PForTesting())
	if err != nil {
		t.Error(err.Error())
		return n, repo.DatasetRef{}
	}

	ref := addCitiesDataset(t, n.Repo)
	return n, ref
}

func testReadDataset(t *testing.T, rmf RepoMakerFunc) {
	ctx := context.Background()
	n, ref := createDataset(t, rmf)

	if err := base.ReadDataset(ctx, n.Repo, &ref); err != nil {
		t.Error(err.Error())
		return
	}

	if ref.Dataset == nil {
		t.Error("expected dataset to not equal nil")
		return
	}
}

func testRenameDataset(t *testing.T, rmf RepoMakerFunc) {
	ctx := context.Background()
	node, ref := createDataset(t, rmf)

	b := &repo.DatasetRef{
		Name:     "cities2",
		Peername: "me",
	}

	if err := base.ModifyDatasetRef(ctx, node.Repo, &ref, b, true); err != nil {
		t.Error(err)
		return
	}

	if err := base.ReadDataset(ctx, node.Repo, b); err != nil {
		t.Error(err)
		return
	}

	if b.Dataset == nil {
		t.Error("expected dataset to not equal nil")
		return
	}
}

func testDeleteDataset(t *testing.T, rmf RepoMakerFunc) {
	ctx := context.Background()
	node, ref := createDataset(t, rmf)

	if err := base.DeleteDataset(ctx, node.Repo, &ref); err != nil {
		t.Error(err.Error())
		return
	}
}
