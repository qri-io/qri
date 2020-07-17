package lib

import (
	"context"
	"testing"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/config"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestWebsocket(t *testing.T) {
	tr, err := repotest.NewTempRepo("foo", "websocket_test", repotest.NewTestCrypto())
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Delete()

	cfg := config.DefaultConfigForTesting()
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

	wsCtx, wsCancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		inst.ServeWebsocket(wsCtx)
		done <- struct{}{}
	}()

	wsCancel()
	<-done
}
