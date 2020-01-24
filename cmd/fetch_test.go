package cmd

import (
	"context"
	"testing"
	"time"
)

func TestFetchCommand(t *testing.T) {
	r := NewTestRunner(t, "peer_a", "qri_test_fetch_a")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	if err != nil {
		t.Fatal(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	if err = executeCommand(cmdR, "qri save --body=testdata/movies/body_thirty.csv me/test_movies"); err != nil {
		t.Fatal(err)
	}

	cmdR = r.CreateCommandRunner(ctx)
	if err = executeCommand(cmdR, "qri log peer_a/test_movies"); err != nil {
		t.Fatal(err)
	}

	text := r.GetCommandOutput()
	// TODO (b5) - make this acutally inspect once we have stable timestamps in logs
	if len(text) == 0 {
		t.Errorf("expected log to produce a non-zero length output.")
	}

	cmdR = r.CreateCommandRunner(ctx)
	if err = executeCommand(cmdR, "qri config set remote.enabled true rpc.enabled false"); err != nil {
		t.Fatal(err)
	}

	cmdR = r.CreateCommandRunner(ctx)
	go func() {
		if err = executeCommand(cmdR, "qri connect"); err != nil {
			t.Fatal(err)
		}
	}()

	// TODO (b5) - this is horrible. we should block on a channel receive for connectedness
	time.Sleep(time.Second * 5)

	b := NewTestRunner(t, "peer_b", "qri_test_fetch_b")
	defer b.Delete()

	cmdBr := b.CreateCommandRunner(ctx)
	if err = executeCommand(cmdBr, "qri log peer_b/test_movies"); err == nil {
		t.Fatal("expected fetch on non-existant log to error")
	}

	cmdBr = b.CreateCommandRunner(ctx)
	if err = executeCommand(cmdBr, "qri config set remotes.a_node http://localhost:2503"); err != nil {
		t.Fatal(err)
	}

	cmdBr = b.CreateCommandRunner(ctx)
	if err = executeCommand(cmdBr, "qri fetch peer_a/test_movies --remote a_node"); err != nil {
		t.Fatal(err)
	}

	cmdBr = b.CreateCommandRunner(ctx)
	if err = executeCommand(cmdBr, "qri logbook --raw"); err != nil {
		t.Fatal(err)
	}

	text = b.GetCommandOutput()
	t.Logf("%s", text)

	cmdBr = b.CreateCommandRunner(ctx)
	if err = executeCommand(cmdBr, "qri log peer_a/test_movies"); err != nil {
		t.Fatal(err)
	}

	text = b.GetCommandOutput()
	t.Logf("expect log: '%s'", text)
}
