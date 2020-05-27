package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

type testRunner struct {
	Ctx context.Context
}

func newTestRunner(t *testing.T) (tr *testRunner, cleanup func()) {
	tr = &testRunner{
		Ctx: context.Background(),
	}

	cleanup = func() {}
	return tr, cleanup
}

func (tr *testRunner) IPFSBackedQriNode(t *testing.T, username string) *QriNode {
	ipfs, _, err := p2ptest.MakeIPFSNode(tr.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	r, err := p2ptest.MakeRepoFromIPFSNode(ipfs, username)
	if err != nil {
		t.Fatal(err)
	}
	node, err := NewQriNode(r, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err)
	}
	return node
}

func writeWorldBankPopulation(ctx context.Context, t *testing.T, r repo.Repo) reporef.DatasetRef {
	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Time{} }
	defer func() { dsfs.Timestamp = prevTs }()

	ds := &dataset.Dataset{
		Name: "world_bank_population",
		Commit: &dataset.Commit{
			Title: "initial commit",
		},
		Meta: &dataset.Meta{
			Title: "World Bank Population",
		},
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
		Viz: &dataset.Viz{
			Format: "html",
		},
		Transform: &dataset.Transform{
			Syntax: "amaze",
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[100]")))

	ref, err := base.CreateDataset(ctx, r, ds, nil, base.SaveSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err)
	}

	return ref
}
