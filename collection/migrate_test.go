package collection

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestMigrateRepoStoreToLocalCollectionSet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, err := repotest.NewTestRepo()
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

	set, err := NewLocalSet(ctx, "", func(o *LocalSetOptions) {
		o.MigrateRepo = r
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := set.List(ctx, r.Profiles().Owner(ctx).ID, params.ListAll)
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(expect, got, cmpopts.IgnoreFields(dsref.VersionInfo{}, "InitID", "MetaTitle", "ThemeList", "BodySize", "BodyRows", "CommitTime", "NumErrors", "CommitTitle")); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
