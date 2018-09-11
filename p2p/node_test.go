package p2p

import (
	"testing"

	"github.com/qri-io/qri/config"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/test"
)

func TestNewNode(t *testing.T) {
	info := cfgtest.GetTestPeerInfo(0)
	r, err := test.NewTestRepoFromProfileID(profile.ID(info.PeerID), 0, -1)
	if err != nil {
		t.Errorf("error creating test repo: %s", err.Error())
		return
	}

	p2pconf := config.DefaultP2PForTesting()
	node, err := NewTestableQriNode(r, p2pconf)
	if err != nil {
		t.Errorf("error creating qri node: %s", err.Error())
		return
	}
	n := node.(*QriNode)
	if n.Online {
		t.Errorf("default node online flag should be false")
	}
	if err := n.Connect(); err != nil {
		t.Error(err.Error())
	}
	if !n.Online {
		t.Errorf("online should equal true")
	}
}
