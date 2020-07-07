package p2ptest

import (
	"context"
	"testing"

	"github.com/qri-io/qri/event"
)

func TestMakeIPFS(t *testing.T) {
	ctx := context.Background()
	if _, _, err := MakeIPFSSwarm(ctx, true, 11); err == nil {
		t.Errorf("expected an error creating more than 10 nodes")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	nodes, _, err := MakeIPFSSwarm(ctx, true, 5)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = MakeRepoFromIPFSNode(ctx, nodes[0], "ramfox", event.NilBus); err != nil {
		t.Fatal(err)
	}
}

func TestMakeIPFSSwarmMockIdentity(t *testing.T) {
	t.Skip("TODO (b5) - causes crash in bootstrap")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, _, err := MakeIPFSSwarm(ctx, false, 5)
	if err != nil {
		t.Fatal(err)
	}
}
