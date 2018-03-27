package p2p

import (
	"fmt"
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	testutil "gx/ipfs/QmWRCn8vruNAzHx8i6SAXinuheRitKEGu8c7m26stKvsYx/go-testutil"
	// peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

func TestNewNode(t *testing.T) {
	pid, err := testutil.RandPeerID()
	if err != nil {
		t.Errorf("error generating peer id: %s", err.Error())
		return
	}

	r, err := NewTestRepo(profile.ID(pid))
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

var repoID = 0

func NewTestRepo(id profile.ID) (repo.Repo, error) {
	repoID++
	return repo.NewMemRepo(&profile.Profile{
		ID:       id,
		Peername: fmt.Sprintf("test-repo-%d", repoID),
	}, cafs.NewMapstore(), profile.MemStore{})
}
