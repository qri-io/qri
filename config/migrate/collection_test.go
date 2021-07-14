package migrate

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/collection"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestMigrateRepoStoreToLocalCollectionSet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempRepo, err := repotest.NewTempRepo("migration_test", "collection_migration_test", repotest.NewTestCrypto())
	if err != nil {
		t.Fatal(err)
	}

	if err := tempRepo.AddDatasets(ctx); err != nil {
		t.Fatal(err)
	}

	r, err := tempRepo.Repo(ctx)
	if err != nil {
		t.Fatal(err)
	}

	expect, err := repo.ListVersionInfoShim(r, 0, 100000)
	if err != nil {
		t.Error(err)
	}

	if len(expect) == 0 {
		t.Fatalf("test repo has no datasets")
	}

	if err := MigrateRepoStoreToLocalCollectionSet(ctx, tempRepo.QriPath, tempRepo.GetConfig()); err != nil {
		t.Fatal(err)
	}

	set, err := collection.NewLocalSet(ctx, event.NilBus, tempRepo.QriPath)
	if err != nil {
		t.Fatal(err)
	}

	proID := profile.IDB58MustDecode(tempRepo.GetConfig().Profile.ID)

	got, err := set.List(ctx, proID, params.ListAll)
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(expect, got, cmpopts.IgnoreFields(dsref.VersionInfo{}, "InitID")); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}

}
