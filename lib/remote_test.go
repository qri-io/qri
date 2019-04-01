package lib

import (
	"math/rand"
	"testing"

	"github.com/qri-io/dag/dsync"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	testrepo "github.com/qri-io/qri/repo/test"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestRemote(t *testing.T) {
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}

	// Set a seed so that the sessionID is deterministic
	rand.Seed(5678)

	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	req := NewRemoteRequests(node, nil)
	req.Receivers = dsync.NewTestReceivers()

	exampleDagInfo := `{"manifest":{"links":[[0,1]],"nodes":["QmAbc123","QmDef678"]},"labels":{"bd":0,"cm":0,"st":0},"sizes":[123]}`

	// Reject all dag.Info's
	Config.API.RemoteAcceptSizeMax = 0
	params := ReceiveParams{
		Body: exampleDagInfo,
	}
	result := ReceiveResult{}
	err = req.Receive(&params, &result)
	if err != nil {
		t.Errorf(err.Error())
	}
	if result.Success {
		t.Errorf("error: expected !result.Success")
	}
	expect := `not accepting any datasets`
	if result.RejectReason != expect {
		t.Errorf("error: expected: \"%s\", got \"%s\"", expect, result.RejectReason)
	}

	// Accept all dag.Info's
	Config.API.RemoteAcceptSizeMax = -1
	params = ReceiveParams{
		Body: exampleDagInfo,
	}
	result = ReceiveResult{}
	err = req.Receive(&params, &result)
	if err != nil {
		t.Errorf(err.Error())
	}
	if !result.Success {
		t.Errorf("error: expected result.Success")
	}
	if result.RejectReason != "" {
		t.Errorf("expected no error, but got \"%s\"", result.RejectReason)
	}
	expect = `CoTeMqzUaa`
	if result.SessionID != expect {
		t.Errorf("expected sessionID to be \"%s\", but got \"%s\"", expect, result.SessionID)
	}
}
