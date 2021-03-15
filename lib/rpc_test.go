package lib

import (
	"context"
	"errors"
	"testing"
)

func TestRPCRequest(t *testing.T) {
	// TODO (b5) - shouldn't need the network test runner for this. use the
	// standard test runner once it uses lib.NewInstance instead of
	// NewInstanceFromConfigAndNode
	tr := NewNetworkIntegrationTestRunner(t, "rpc")
	defer tr.Cleanup()

	adnanInst := tr.InitAdnan(t)

	ref := InitWorldBankDataset(tr.Ctx, t, adnanInst)

	repoLockCtx, closeAdnanInst := context.WithCancel(context.Background())
	defer closeAdnanInst()
	go adnanInst.ServeRPC(repoLockCtx)

	rpcCtx, closeAdnanRPC := context.WithCancel(context.Background())
	defer closeAdnanRPC()

	// another node is connected
	adnanRPCInst, err := NewInstance(rpcCtx, tr.adnanRepo.QriPath)
	if err != nil {
		t.Fatal(err)
	}

	_, err = adnanInst.Dataset().Get(tr.Ctx, &GetParams{
		Refstr: ref.String(),
	})

	if err != nil {
		t.Error(err)
	}

	if err := <-adnanRPCInst.Shutdown(); err != nil {
		t.Error(err)
	}
	if err := <-adnanInst.Shutdown(); err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
	}
}
