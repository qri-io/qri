package p2p

import (
	"context"
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/base"
	p2ptest "github.com/qri-io/qri/p2p/test"
)

func TestRequestLogDiff(t *testing.T) {
	ctx := context.Background()
	streams := ioes.NewDiscardIOStreams()
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestDirNetwork(ctx, factory)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	peers := asQriNodes(testPeers)

	tc, err := dstest.NewTestCaseFromDir("testdata/tim/craigslist")
	if err != nil {
		t.Fatal(err)
	}

	// add a dataset to peer 4
	ref, _, err := base.CreateDataset(peers[4].Repo, streams, tc.Name, tc.Input, tc.BodyFile(), false, true)
	if err != nil {
		t.Fatal(err)
	}

	// simulate IPFS connection (we're not on IPFS when running tests)
	peers[3].Repo.Store().(*cafs.MapStore).AddConnection(peers[4].Repo.Store().(*cafs.MapStore))

	// give that dataset to peer 3
	if err := base.FetchDataset(peers[3].Repo, &ref, true, false); err != nil {
		t.Fatalf("error fetching dataset: %s", err)
	}

	prevTitle := tc.Input.Meta.Title
	tc.Input.Meta.Title = "update"
	tc.Input.PreviousPath = ref.Path
	defer func() {
		// because test cases are cached for performance, we need to clean up any mutation to
		// testcase input
		tc.Input.Meta.Title = prevTitle
		tc.Input.PreviousPath = ""
	}()

	// add an update on peer 4
	ref2, _, err := base.CreateDataset(peers[4].Repo, streams, tc.Name, tc.Input, tc.BodyFile(), false, true)
	if err != nil {
		t.Fatal(err)
	}

	ldr, err := peers[3].RequestLogDiff(&ref)
	if err != nil {
		t.Error(err)
	}

	if !ldr.Head.Equal(ref2) {
		t.Errorf("wrong head sent")
	}
}
