package actions

import (
	"testing"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestDatasetLog(t *testing.T) {
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
	log, err := DatasetLog(node, ref, -1, 0)
	if err == nil {
		t.Errorf("expected lookup for nonexistent log to fail")
	}

	ref = repo.MustParseDatasetRef("peer/movies")
	if log, err = DatasetLog(node, ref, -1, 0); err != nil {
		t.Error(err.Error())
	}
	if len(log) != 1 {
		t.Errorf("log length mismatch. expected: %d, got: %d", 1, len(log))
	}

}
