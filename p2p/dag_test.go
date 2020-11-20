package p2p

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dag"
	p2ptest "github.com/qri-io/qri/p2p/test"
)

func TestNewManifest(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	node := tr.IPFSBackedQriNode(t, "dag_tests_peer")
	ref := writeWorldBankPopulation(tr.Ctx, t, node.Repo)

	mfst, err := node.NewManifest(tr.Ctx, ref.Path)
	if err != nil {
		t.Error(err)
	}

	expect := &dag.Manifest{
		Links: [][2]int{{0, 1}, {0, 2}, {0, 3}, {0, 4}, {0, 5}, {0, 6}, {0, 7}},
		Nodes: []string{
			"QmXrEEWgrTgLjDG4PyJZFmeGH8gvf49Hi6JVC2r4L1JJbW",
			"QmPtfTWctksdUXfXCmpQf7pF8z5xUNAYrurQxY5LRsKrX1",
			"QmQYhY2RaVLn36MrjLQG9XJk6QQMHkSvvPUXZ9J9MDf6Ur",
			"QmTgqZXtLnU2nRU4yMaQKBiMPesavuDVCfBWJgDvbQZ2xm",
			"QmWVxUKnBmbiXai1Wgu6SuMzyZwYRqjt5TXL8xxghN5hWL",
			"Qma3bmcJhAdKeEB9dKJBfChVb2LvcNfWvqnh7hqbJR7aLZ",
			"QmaeD2dPHLJsM2pyTGMmbuSAj26rHK8DvoSuY3s2LFutFe",
			"QmdzHjr5GdFGCvo9dCqdhUpqPxA6x5yz8G1cErb7q5MvTP",
		},
	}

	if diff := cmp.Diff(expect, mfst); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}

func TestMissingManifest(t *testing.T) {
	// TODO (b5) - we're running into network fetching issues here, the generated
	// ipfs node isn't currently creating a localNodeGetter, causing this test
	// to hang forever trying to fetch on a one-node network
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	node := tr.IPFSBackedQriNode(t, "dag_tests_peer")
	ref := writeWorldBankPopulation(tr.Ctx, t, node.Repo)

	// Select some blocks from the saved dataset. Don't hardcode block ids, because if they
	// ever change this test will hang.
	capi, _ := node.IPFSCoreAPI()
	blocks := p2ptest.GetSomeBlocks(capi, ref, 2)
	in := &dag.Manifest{Nodes: blocks}

	// TODO(dlong): This function seems to not work correctly. If any blocks are missing, it
	// doesn't return them, instead it hangs forever.

	// Get which blocks from the manifest are missing from available blocks.
	mfst, err := node.MissingManifest(tr.Ctx, in)
	if err != nil {
		t.Error(err)
	}

	// None of those blocks in the manifest are missing.
	expect := &dag.Manifest{}
	if diff := cmp.Diff(expect, mfst); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}

func TestNewDAGInfo(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	node := tr.IPFSBackedQriNode(t, "dag_tests_peer")
	ref := writeWorldBankPopulation(tr.Ctx, t, node.Repo)
	di, err := node.NewDAGInfo(tr.Ctx, ref.Path, "")
	if err != nil {
		t.Error(err)
	}

	expect := &dag.Info{
		Labels: map[string]int{
			"bd": 4,
			"cm": 1,
			"md": 5,
			"st": 3,
			"sa": 2,
		},
		Sizes: []uint64{1710, 472, 157, 166, 13, 54, 428, 39},
		Manifest: &dag.Manifest{
			Links: [][2]int{{0, 1}, {0, 2}, {0, 3}, {0, 4}, {0, 5}, {0, 6}, {0, 7}},
			Nodes: []string{
				"QmXrEEWgrTgLjDG4PyJZFmeGH8gvf49Hi6JVC2r4L1JJbW",
				"QmPtfTWctksdUXfXCmpQf7pF8z5xUNAYrurQxY5LRsKrX1",
				"QmQYhY2RaVLn36MrjLQG9XJk6QQMHkSvvPUXZ9J9MDf6Ur",
				"QmTgqZXtLnU2nRU4yMaQKBiMPesavuDVCfBWJgDvbQZ2xm",
				"QmWVxUKnBmbiXai1Wgu6SuMzyZwYRqjt5TXL8xxghN5hWL",
				"Qma3bmcJhAdKeEB9dKJBfChVb2LvcNfWvqnh7hqbJR7aLZ",
				"QmaeD2dPHLJsM2pyTGMmbuSAj26rHK8DvoSuY3s2LFutFe",
				"QmdzHjr5GdFGCvo9dCqdhUpqPxA6x5yz8G1cErb7q5MvTP",
			},
		},
	}

	if diff := cmp.Diff(expect, di); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}
