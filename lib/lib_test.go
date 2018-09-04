package lib

import (
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestReceivers(t *testing.T) {
	r, err := repo.NewMemRepo(&profile.Profile{}, cafs.NewMapstore(), profile.NewMemStore(), nil)
	if err != nil {
		t.Errorf("error creating mem repo: %s", err)
		return
	}
	n, err := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode).New(r)
	if err != nil {
		t.Errorf("error creating qri node: %s", err)
		return
	}

	node := n.(*p2p.QriNode)
	reqs := Receivers(node)
	if len(reqs) != 8 {
		t.Errorf("unexpected number of receivers returned. expected: %d. got: %d\nhave you added/removed a receiver?", 8, len(reqs))
		return
	}
}
