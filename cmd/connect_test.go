package cmd

import (
	"context"
	"testing"
	"time"

	regmock "github.com/qri-io/qri/registry/regserver"
)

func TestConnect(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// Setup the test repo so that connect can run
	run := NewTestRunner(t, "test_peer", "qri_test_connect")
	defer run.Delete()

	// Construct a mock registry to pass to the connect command
	_, registryServer := regmock.NewMockServer()

	// Configure ports such that other tests do not conflict with the connection ports
	// TODO(dustmop): Add websocket.port to config, set that here
	run.MustExec(t, "qri config set api.port 9871 rpc.port 9872")

	cmd := "qri connect --registry=" + registryServer.URL

	// Run the command for 1 second
	ctx, done := context.WithTimeout(context.Background(), time.Second)
	defer done()

	defer func() {
		if e := recover(); e != nil {
			t.Errorf("unexpected panic:\n%s\n%s", cmd, e)
			return
		}
	}()

	err := run.ExecCommandWithContext(ctx, cmd)
	if err != nil {
		t.Errorf("unexpected error executing command\n%s\n%s", cmd, err.Error())
		return
	}
}
