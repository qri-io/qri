package cmd

import (
	"context"
	"testing"

	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestPreviewCommand(t *testing.T) {
	run := NewTestRunner(t, "test_peer_preview_command", "qri_test_preview_command")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())

	// TODO(dustmop): Move into test runner
	reg, cleanup, err := regserver.NewTempRegistry(ctx, "temp_registry", "", repotest.NewTestCrypto())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	// TODO(b5): need to defer in this order. the deferred cleanup command blocks on done,
	// which is in turn blocked on cancel. deferring in the other order deadlocks.
	// the smarter way to deal with this is to refactor TempRegistry to use the Done pattern
	defer cancel()

	// Create a mock registry, point our test runner to its URL
	_, httpServer := regserver.NewMockServerRegistry(*reg)
	run.RepoRoot.GetConfig().Registry.Location = httpServer.URL
	err = run.RepoRoot.WriteConfigFile()
	if err != nil {
		t.Fatal(err)
	}

	// Save one commit
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv --file testdata/movies/meta_override.yaml --file testdata/movies/about_movies.md me/movies_preview_test")

	// Publish to the registry
	run.MustExec(t, "qri publish test_peer_preview_command/movies_preview_test")
	run.MustExec(t, "qri delete --all me/movies_preview_test")

	// Search, verify that we get the dataset back
	results, err := reg.Search.Search(registry.SearchParams{
		Q:      "",
		Limit:  2,
		Offset: 0,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Errorf("expected: 1 result, got %d results", len(results))
	}

	result := run.MustExecCombinedOutErr(t, "qri preview test_peer_preview_command/movies_preview_test")
	t.Log(result)
}
