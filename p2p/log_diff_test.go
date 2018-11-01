package p2p

import (
	"testing"
)

func TestRequestLogDiff(t *testing.T) {
	// ctx := context.Background()
	// streams := ioes.NewDiscardIOStreams()
	// factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	// testPeers, err := p2ptest.NewTestDirNetwork(ctx, factory)
	// if err != nil {
	// 	t.Fatalf("error creating network: %s", err.Error())
	// }
	// if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
	// 	t.Fatalf("error connecting peers: %s", err.Error())
	// }

	// peers := asQriNodes(testPeers)

	// tc, err := dstest.NewTestCaseFromDir("testdata/tim/craigslist")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // add a dataset to tim
	// ref, _, err := base.CreateDataset(peers[4].Repo, streams, tc.Name, tc.Input, tc.BodyFile(), false, true)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// //
	// // peers[]

	// prevTitle := tc.Input.Meta.Title
	// tc.Input.Meta.Title = "update"
	// tc.Input.PreviousPath = ref.Path
	// defer func() {
	// 	// because test cases are cached for performance, we need to clean up any mutation to
	// 	// testcase input
	// 	tc.Input.Meta.Title = prevTitle
	// 	tc.Input.PreviousPath = ref.Path
	// }()

	// ref2, _, err := base.CreateDataset(peers[4].Repo, streams, tc.Name, tc.Input, tc.BodyFile(), false, true)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// t.Logf("testing RequestDatasetLog message with %d peers", len(peers))
	// // var wg sync.WaitGroup
	// for i, p1 := range peers {
	// 	for _, p2 := range peers[i+1:] {
	// 		// TODO - having these in parallel is causing races when encoding logs
	// 		// wg.Add(1)
	// 		// go func(p1, p2 *QriNode) {
	// 		// 	defer wg.Done()

	// 		refs, err := p1.RequestDatasetLog(ref, 100, 0)
	// 		if err != nil {
	// 			t.Errorf("%s -> %s error: %s", p1.ID.Pretty(), p2.ID.Pretty(), err.Error())
	// 		}
	// 		if refs == nil {
	// 			t.Error("profile shouldn't be nil")
	// 			return
	// 		}
	// 		// t.Log(refs)
	// 		// }(p1, p2)
	// 	}
	// }

	// wg.Wait()
}
