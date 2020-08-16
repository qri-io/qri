package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	regmock "github.com/qri-io/qri/registry/regserver"
)

func TestConnect(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// Setup the test repo so that connect can run
	run := NewTestRunner(t, "test_peer_qri_test_connect", "qri_test_connect")
	defer run.Delete()

	// Construct a mock registry to pass to the connect command
	_, registryServer := regmock.NewMockServer()

	// TODO(b5): this is supposed to free up ports, but locks when not completely disabled:
	run.MustExec(t, "qri config set api.enabled false rpc.enabled false")
	// Configure ports such that other tests do not conflict with the connection ports
	// run.MustExec(t, "qri config set api.address /ip4/127.0.0.1/tcp/0 api.websocketaddress /ip4/127.0.0.1/tcp/0 rpc.address /ip4/127.0.0.1/tcp/0")

	u, _ := url.Parse(registryServer.URL)

	cmd := "qri connect --registry=" + fmt.Sprintf("/ip4/127.0.0.1/tcp/%s", strings.Split(u.Host, ":")[1])

	defer func() {
		if e := recover(); e != nil {
			t.Errorf("unexpected panic:\n%s\n%s", cmd, e)
			return
		}
	}()

	// Run the command for 1 second
	ctx, done := context.WithTimeout(context.Background(), time.Second)
	defer done()

	err := run.ExecCommandWithContext(ctx, cmd)
	if err != nil {
		t.Errorf("unexpected error executing command\n%s\n%s", cmd, err.Error())
		return
	}
	run.Delete()
}
