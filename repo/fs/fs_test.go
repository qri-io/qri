package fsrepo

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/test/spec"
)

func TestRepo(t *testing.T) {
	path, err := ioutil.TempDir("", "qri_repo_test")
	if err != nil {
		t.Fatal(err)
	}

	rmf := func(t *testing.T) (repo.Repo, func()) {
		if err := os.RemoveAll(path); err != nil {
			t.Fatalf("error removing files: %q", err)
		}

		pro, err := profile.NewProfile(config.DefaultProfileForTesting())
		if err != nil {
			t.Fatal(err)
		}

		fs := qfs.NewMemFS()
		book, err := logbook.NewJournal(pro.PrivKey, pro.Peername, fs, path)
		if err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()
		cache := dscache.NewDscache(ctx, fs, nil, pro.Peername, "")

		store := cafs.NewMapstore()
		r, err := NewRepo(store, fs, book, cache, pro, path)
		if err != nil {
			t.Fatalf("error creating repo: %s", err.Error())
		}

		cleanup := func() {
			if err := os.RemoveAll(path); err != nil {
				t.Errorf("error cleaning up after test: %s", err)
			}
		}

		return r, cleanup
	}

	spec.RunRepoTests(t, rmf)

	if err := os.RemoveAll(path); err != nil {
		t.Errorf("error cleaning up after test: %s", err.Error())
	}
}

func TestResolveRef(t *testing.T) {
	path, err := ioutil.TempDir("", "qri_repo_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(path)

	pro, err := profile.NewProfile(config.DefaultProfileForTesting())
	if err != nil {
		t.Fatal(err)
	}

	fs := qfs.NewMemFS()
	book, err := logbook.NewJournal(pro.PrivKey, pro.Peername, fs, path)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	cache := dscache.NewDscache(ctx, fs, nil, "", "")

	store := cafs.NewMapstore()
	r, err := NewRepo(store, fs, book, cache, pro, path)
	if err != nil {
		t.Fatalf("error creating repo: %s", err.Error())
	}

	dsrefspec.AssertResolverSpec(t, r, func(ref dsref.Ref, author identity.Author, log *oplog.Log) error {
		return r.Logbook().MergeLog(ctx, author, log)
	})
}
