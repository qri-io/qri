package p2p

import (
	"fmt"
	"testing"

	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

func TestNewNode(t *testing.T) {
	r, err := NewTestRepo("foo")
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

func NewTestRepo(id peer.ID) (repo.Repo, error) {
	repoID++
	return repo.NewMemRepo(&profile.Profile{
		ID:       id.Pretty(),
		Peername: fmt.Sprintf("test-repo-%d", repoID),
	}, cafs.NewMapstore(), repo.MemPeers{}, &analytics.Memstore{})
}
