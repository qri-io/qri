package lib

import (
	"context"
	"testing"

	"github.com/qri-io/qfs"
	testcfg "github.com/qri-io/qri/config/test"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestWebsocket(t *testing.T) {
	tr, err := repotest.NewTempRepo("foo", "websocket_test", repotest.NewTestCrypto())
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Delete()

	cfg := testcfg.DefaultConfigForTesting()
	cfg.Filesystems = []qfs.Config{
		{Type: "mem"},
		{Type: "local"},
	}
	cfg.Repo.Type = "mem"

	instCtx, instCancel := context.WithCancel(context.Background())
	defer instCancel()

	inst, err := NewInstance(instCtx, tr.QriPath, OptConfig(cfg))
	if err != nil {
		t.Fatal(err)
	}

	subsCount := inst.bus.NumSubscribers()

	wsCtx, wsCancel := context.WithCancel(context.Background())
	_, err = NewWebsocketHandler(wsCtx, inst)
	if err != nil {
		t.Fatal(err)
	}

	// websockets should subscribe 2 new handlers: FS Watcher & the WS message handler
	if inst.bus.NumSubscribers() != subsCount+2 {
		t.Fatalf("failed to subscribe websocket handlers")
	}

	wsCancel()
}
