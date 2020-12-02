package fsrepo

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
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

		ctx := context.Background()
		bus := event.NewBus(ctx)
		fs, err := muxfs.New(ctx, []qfs.Config{
			{Type: "mem"},
			{Type: "local"},
		})
		if err != nil {
			t.Fatal(err)
		}

		book, err := logbook.NewJournal(pro.PrivKey, pro.Peername, bus, fs, "/mem/logbook.qfb")
		if err != nil {
			t.Fatal(err)
		}

		cache := dscache.NewDscache(ctx, fs, bus, pro.Peername, "")

		r, err := NewRepo(path, fs, book, cache, pro, bus)
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

	ctx := context.Background()
	bus := event.NewBus(ctx)
	fs, err := muxfs.New(ctx, []qfs.Config{
		{Type: "mem"},
		{Type: "local"},
	})
	if err != nil {
		t.Fatal(err)
	}

	book, err := logbook.NewJournal(pro.PrivKey, pro.Peername, bus, fs, "/mem/logbook.qfb")
	if err != nil {
		t.Fatal(err)
	}

	cache := dscache.NewDscache(ctx, fs, bus, "", "")

	r, err := NewRepo(path, fs, book, cache, pro, bus)
	if err != nil {
		t.Fatalf("error creating repo: %s", err.Error())
	}

	dsrefspec.AssertResolverSpec(t, r, func(ref dsref.Ref, author profile.Author, log *oplog.Log) error {
		datasetRef := reporef.RefFromDsref(ref)
		err := r.PutRef(datasetRef)
		if err != nil {
			t.Fatal(err)
		}
		return r.Logbook().MergeLog(ctx, author, log)
	})
}
