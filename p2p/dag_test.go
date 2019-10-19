package p2p

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dag"
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
			"QmdShFSjhU6K96FLEtu1Zm5Wq2bG9avbUcKjEwm84EpXjy",
			"QmQoNqKXP7aZJWS6GLJMx8Ax85uBFpRbmg7Npd6usx5V82",
			"QmRwendrWJkquHoJfeCu3FvCraaWB4qXJ6N5Xv5xvv2qVv",
			"QmVC2qgWvgS4UKvifAkB6rtHBks4vRTw64VmsceeBQ2V3V",
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
	writeWorldBankPopulation(tr.Ctx, t, node.Repo)

	in := &dag.Manifest{
		Nodes: []string{
			"QmdShFSjhU6K96FLEtu1Zm5Wq2bG9avbUcKjEwm84EpXjy", // block from world bank pop DAG
			"Qma3bmcJhAdKeEB9dKJBfChVb2LvcNfWvqnh7hqbJR7aLZ", // block from world bank pop DAG
			// "QmTFauExutTsy4XP6JbMFcw2Wa9645HJt2bTqL6qYDCKfe", // random hash from somewhere else
		},
	}

	mfst, err := node.MissingManifest(tr.Ctx, in)
	if err != nil {
		t.Error(err)
	}

	expect := &dag.Manifest{
		// Nodes: []string{
		// 	"extraHash",
		// },
	}

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
		Sizes: []uint64{0x0602, 0x26, 0x019c, 0x01bc, 0x0d, 0x36, 0x27, 0xa0},
		Manifest: &dag.Manifest{
			Links: [][2]int{{0, 1}, {0, 2}, {0, 3}, {0, 4}, {0, 5}, {0, 6}, {0, 7}},
			Nodes: []string{
				"QmdShFSjhU6K96FLEtu1Zm5Wq2bG9avbUcKjEwm84EpXjy",
				"QmQoNqKXP7aZJWS6GLJMx8Ax85uBFpRbmg7Npd6usx5V82",
				"QmRwendrWJkquHoJfeCu3FvCraaWB4qXJ6N5Xv5xvv2qVv",
				"QmVC2qgWvgS4UKvifAkB6rtHBks4vRTw64VmsceeBQ2V3V",
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
