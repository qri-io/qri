package regserver

import (
	"context"
	"testing"

	repotest "github.com/qri-io/qri/repo/test"
)

func TestMockServer(t *testing.T) {
	NewMockServer()
	NewMockServerRegistry(NewMemRegistry(nil))
}

func TestTempRegistry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gen := repotest.NewTestCrypto()
	_, cleanup, err := NewTempRegistry(ctx, "registry", "regserver_new_temp", gen)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	cancel()
}
