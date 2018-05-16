package p2p

import (
	"testing"

	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/test"

	testutil "gx/ipfs/QmWRCn8vruNAzHx8i6SAXinuheRitKEGu8c7m26stKvsYx/go-testutil"
)

func TestNewNode(t *testing.T) {
	pid, err := testutil.RandPeerID()
	if err != nil {
		t.Errorf("error generating peer id: %s", err.Error())
		return
	}

	r, err := test.NewTestRepoFromProfileID(profile.ID(pid), 0, -1)
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
