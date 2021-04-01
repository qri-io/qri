package lib

import (
	"context"
	"testing"

	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/p2p"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestChanges(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)

	InitWorldBankDataset(ctx, t, inst)
	commitref := Commit2WorldBank(ctx, t, inst)

	// test ChangeReport with one param
	p := &ChangeReportParams{
		RightRefstr: commitref.Alias(),
	}
	if _, err := inst.Diff().Changes(ctx, p); err != nil {
		t.Fatalf("change report error: %s", err)
	}
}
