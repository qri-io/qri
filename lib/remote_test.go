package lib

import (
	"testing"

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

	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	req := NewRemoteRequests(node, nil)

	var rejectReason string

	// Reject all dag.Info's
	Config.API.RemoteAcceptSizeMax = 0
	params := ReceiveParams{
		Body: "{\"Sizes\":[10,20,30]}",
	}
	err = req.Receive(&params, &rejectReason)
	if err != nil {
		t.Errorf(err.Error())
	}
	expect := `not accepting any datasets`
	if rejectReason != expect {
		t.Errorf("error: expected: \"%s\", got \"%s\"", expect, rejectReason)
	}

	// Accept all dag.Info's
	Config.API.RemoteAcceptSizeMax = -1
	params = ReceiveParams{
		Body: "{\"Sizes\":[10,20,30]}",
	}
	err = req.Receive(&params, &rejectReason)
	if err != nil {
		t.Errorf(err.Error())
	}
	if rejectReason != "" {
		t.Errorf("expected no error, but got \"%s\"", rejectReason)
	}

}
