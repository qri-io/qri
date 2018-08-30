package p2p

import (
	"testing"

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

	node, err := NewQriNode(r)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		return
	}
	if node.Online != true {
		t.Errorf("default node online flag should be true")
	}
}
