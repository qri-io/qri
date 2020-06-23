package cmd

import (
	"context"
	"testing"

	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver"
	repotest "github.com/qri-io/qri/repo/test"
)

// Test publishing to a mock registry
func TestPublish(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_registry_publish")
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
	run.MustExec(t, "qri save me/one_ds --body testdata/movies/body_ten.csv")

	// Publish to the registry
	run.MustExec(t, "qri publish me/one_ds")

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
	if results[0].Name != "one_ds" {
		t.Errorf("expected: dataset named \"one_ds\", got %q", results[0].Name)
	}
}
