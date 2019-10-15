package cmd

import (
	"context"
	"testing"
)

func TestLogbookCommand(t *testing.T) {
	r := NewTestRepoRoot(t, "qri_test_logbook")
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
	if err = executeCommand(cmdR, "qri logbook me/test_movies --raw"); err == nil {
		t.Error("expected using a ref and the raw flag to error")
	}

	cmdR = r.CreateCommandRunner(ctx)
	if err = executeCommand(cmdR, "qri logbook --raw"); err != nil {
		t.Fatal(err)
	}

	// TODO (b5) - once we have consistent timestamps from the logbook package
	// for testing, enable this & compare output
	// res := r.GetOutput()
	// t.Log(res)

	cmdR = r.CreateCommandRunner(ctx)
	if err = executeCommand(cmdR, "qri logbook me/test_movies"); err != nil {
		t.Fatal(err)
	}
}
