package lib

// func TestRemote(t *testing.T) {
// 	cfg := config.DefaultConfigForTesting()
// 	rc, _ := regmock.NewMockServer()
// 	mr, err := testrepo.NewTestRepo(rc)
// 	if err != nil {
// 		t.Fatalf("error allocating test repo: %s", err.Error())
// 	}

// 	// Set a seed so that the sessionID is deterministic
// 	rand.Seed(5678)

// 	node, err := p2p.NewQriNode(mr, cfg.P2P)
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}

// 	inst := &Instance{ node: node, cfg: cfg }
// 	req := NewRemoteMethods(inst)
// 	req.Receivers = dsync.NewTestReceivers()

// 	exampleDagInfo := &dag.Info{
// 		Manifest: &dag.Manifest{
// 			Links: [][2]int{{0, 1}},
// 			Nodes: []string{"QmAbc123", "QmDef678"},
// 		},
// 		Labels: map[string]int{"bd": 0, "cm": 0, "st": 0},
// 		Sizes:  []uint64{123},
// 	}

// 	// Reject all dag.Info's
// 	cfg.API.RemoteAcceptSizeMax = 0
// 	params := ReceiveParams{
// 		DagInfo: exampleDagInfo,
// 	}
// 	result := ReceiveResult{}
// 	err = req.Receive(&params, &result)
// 	if err != nil {
// 		t.Errorf(err.Error())
// 	}
// 	if result.Success {
// 		t.Errorf("error: expected !result.Success")
// 	}
// 	expect := `not accepting any datasets`
// 	if result.RejectReason != expect {
// 		t.Errorf("error: expected: \"%s\", got \"%s\"", expect, result.RejectReason)
// 	}

// 	// Accept all dag.Info's
// 	cfg.API.RemoteAcceptSizeMax = -1
// 	params = ReceiveParams{
// 		DagInfo: exampleDagInfo,
// 	}
// 	result = ReceiveResult{}
// 	err = req.Receive(&params, &result)
// 	if err != nil {
// 		t.Errorf(err.Error())
// 	}
// 	if !result.Success {
// 		t.Errorf("error: expected result.Success")
// 	}
// 	if result.RejectReason != "" {
// 		t.Errorf("expected no error, but got \"%s\"", result.RejectReason)
// 	}
// 	expect = `CoTeMqzUaa`
// 	if result.SessionID != expect {
// 		t.Errorf("expected sessionID to be \"%s\", but got \"%s\"", expect, result.SessionID)
// 	}
// }
