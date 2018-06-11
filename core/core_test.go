package core

import (
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

const (
	// numReceivers = 6
	numReceivers = 7
)

func TestReceivers(t *testing.T) {
	node, err := testQriNode()
	if err != nil {
		t.Errorf("error creating qri node: %s", err.Error())
		return
	}

	reqs := Receivers(node)
	if len(reqs) != numReceivers {
		t.Errorf("unexpected number of receivers returned. expected: %d. got: %d\nhave you added/removed a receiver?", numReceivers, len(reqs))
		return
	}
}

func testQriNode(cfgs ...func(c *config.P2P)) (*p2p.QriNode, error) {
	r, err := repo.NewMemRepo(&profile.Profile{}, cafs.NewMapstore(), profile.NewMemStore(), nil)
	if err != nil {
		return nil, err
	}

	return p2p.NewQriNode(r, func(c *config.P2P) {
		c.Enabled = false
		for _, cfg := range cfgs {
			cfg(c)
		}
	})
}
