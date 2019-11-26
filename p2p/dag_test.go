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
			"QmTxSFyAAq7xa7L5DKAC1rYq1jR3GrAdpaTfMgxd2HU2Sw",
			"QmQoNqKXP7aZJWS6GLJMx8Ax85uBFpRbmg7Npd6usx5V82",
			"QmSbMMHCQ2wetbXJsviNdfKkELS6jxLaHeqxdzT6DAmVZU",
			"QmW27MUFMSvPiE3FpmHhSeBZQEuYAppofudDCLvPXVfSLR",
			"QmWVxUKnBmbiXai1Wgu6SuMzyZwYRqjt5TXL8xxghN5hWL",
			"Qma3bmcJhAdKeEB9dKJBfChVb2LvcNfWvqnh7hqbJR7aLZ",
			"QmdzHjr5GdFGCvo9dCqdhUpqPxA6x5yz8G1cErb7q5MvTP",
			"QmewFt8f53Do9hCKTD76MyBpi19WJkoCqkC96VGnbKd5Ak",
		},
	}

	if diff := cmp.Diff(expect, mfst); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}

func TestMissingManifest(t *testing.T) {
	// TODO (b5) - we're running into network fetching issues here, the generated
	// ipts node isn't currently creating a localNodeGetter, causing this test
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
			"cm": 3,
			"md": 5,
			"st": 7,
			"tf": 6,
			"vz": 1,
		},
		Sizes: []uint64{0x061e, 0x26, 0x019c, 0x01d8, 0x0d, 0x36, 0x27, 0xa0},
		Manifest: &dag.Manifest{
			Links: [][2]int{{0, 1}, {0, 2}, {0, 3}, {0, 4}, {0, 5}, {0, 6}, {0, 7}},
			Nodes: []string{
				"QmTxSFyAAq7xa7L5DKAC1rYq1jR3GrAdpaTfMgxd2HU2Sw",
				"QmQoNqKXP7aZJWS6GLJMx8Ax85uBFpRbmg7Npd6usx5V82",
				"QmSbMMHCQ2wetbXJsviNdfKkELS6jxLaHeqxdzT6DAmVZU",
				"QmW27MUFMSvPiE3FpmHhSeBZQEuYAppofudDCLvPXVfSLR",
				"QmWVxUKnBmbiXai1Wgu6SuMzyZwYRqjt5TXL8xxghN5hWL",
				"Qma3bmcJhAdKeEB9dKJBfChVb2LvcNfWvqnh7hqbJR7aLZ",
				"QmdzHjr5GdFGCvo9dCqdhUpqPxA6x5yz8G1cErb7q5MvTP",
				"QmewFt8f53Do9hCKTD76MyBpi19WJkoCqkC96VGnbKd5Ak",
			},
		},
	}

	if diff := cmp.Diff(expect, di); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}
