package actions

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestDatasetLog(t *testing.T) {
	ctx := context.Background()
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}

	p, err := p2p.NewTestableQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	node := p.(*p2p.QriNode)

	ref := repo.MustParseDatasetRef("peer/not_a_dataset")
	log, err := DatasetLog(ctx, node, ref, -1, 0)
	if err == nil {
		t.Errorf("expected lookup for nonexistent log to fail")
	}

	ref = repo.MustParseDatasetRef("peer/movies")
	if log, err = DatasetLog(ctx, node, ref, 1, 0); err != nil {
		t.Error(err.Error())
	}
	if len(log) != 1 {
		t.Errorf("log length mismatch. expected: %d, got: %d", 1, len(log))
	}

	expect := []base.DatasetLogItem{
		{
			Ref: dsref.Ref{
				Username:  "peer",
				Name:      "movies",
				ProfileID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
				// TODO (b5) - use constant time to make timestamp & path comparable
				Path: "/map/QmfDpSrzqrSM9PctPqDserHRTAaGHUjLLqzYrGEKawU4iN",
			},
			CommitTitle: "initial commit",
		},
	}

	if diff := cmp.Diff(expect, log, cmpopts.IgnoreFields(base.DatasetLogItem{}, "Timestamp"), cmpopts.IgnoreFields(dsref.Ref{}, "Path")); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}

}
