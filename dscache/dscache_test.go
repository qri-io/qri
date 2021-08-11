package dscache

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/localfs"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/profile"
)

// TODO(dlong): Test NewDscache, IsEmpty, Assign, ListRefs, Update

func TestNilCallable(t *testing.T) {
	var (
		cache *Dscache
		err   error
	)

	if !cache.IsEmpty() {
		t.Errorf("expected IsEmpty: got !IsEmpty")
	}
	if err = cache.Assign(&Dscache{}); err != ErrNoDscache {
		t.Errorf("expected '%s': got '%s'", ErrNoDscache, err)
	}
	if str := cache.VerboseString(true); !strings.Contains(str, "empty dscache") {
		t.Errorf("expected str to Contain 'empty dscache': got '%s'", str)
	}
	if _, err = cache.ListRefs(); err != ErrNoDscache {
		t.Errorf("expected '%s': got '%s'", ErrNoDscache, err)
	}
}

func TestDscacheAssignSaveAndLoad(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	ctx := context.Background()
	fs, err := localfs.NewFS(nil)
	if err != nil {
		t.Errorf("error creating local filesystem")
		return
	}

	keyData := testkeys.GetKeyData(0)
	peername := "test_user"

	// Construct a dscache, will not save without a filename
	builder := NewBuilder()
	builder.AddUser(peername, profile.IDFromPeerID(keyData.PeerID).Encode())
	builder.AddDsVersionInfo(dsref.VersionInfo{InitID: "abcd1"})
	builder.AddDsVersionInfo(dsref.VersionInfo{InitID: "efgh2"})
	constructed := builder.Build()

	// A dscache that will save when it is assigned
	dscacheFile := filepath.Join(tmpdir, "dscache.qfb")
	saveable := NewDscache(ctx, fs, event.NilBus, peername, dscacheFile)
	saveable.Assign(constructed)

	// Load the dscache from its serialized file, verify it has correct data
	loadable := NewDscache(ctx, fs, event.NilBus, peername, dscacheFile)
	if loadable.Root.UsersLength() != 1 {
		t.Errorf("expected, 1 user, got %d users", loadable.Root.UsersLength())
	}
	if loadable.Root.RefsLength() != 2 {
		t.Errorf("expected, 2 refs, got %d refs", loadable.Root.RefsLength())
	}
}

func TestResolveRef(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	ctx := context.Background()
	fs, err := localfs.NewFS(nil)
	if err != nil {
		t.Errorf("error creating local filesystem: %s", err)
	}
	path := filepath.Join(tmpdir, "dscache.qfb")
	dsc := NewDscache(ctx, fs, event.NilBus, "test_resolve_ref_user", path)

	dsrefspec.AssertResolverSpec(t, dsc, func(r dsref.Ref, author *profile.Profile, _ *oplog.Log) error {
		builder := NewBuilder()
		profileID := author.ID.Encode()
		builder.AddUser(r.Username, profileID)
		if err != nil {
			return err
		}
		builder.AddDsVersionInfo(dsref.VersionInfo{
			Username:  r.Username,
			InitID:    r.InitID,
			Path:      r.Path,
			ProfileID: profileID,
			Name:      r.Name,
		})
		cache := builder.Build()
		dsc.Assign(cache)
		return nil
	})
}

func TestCacheRefConsistency(t *testing.T) {
	ctx := context.Background()

	fsys := qfs.NewMemFS()

	localUsername := "local_user"
	localDsName := "local_dataset"
	pro, err := profile.NewSparsePKProfile(localUsername, testkeys.GetKeyData(0).PrivKey)
	if err != nil {
		t.Fatal(err)
	}
	book, err := logbook.NewJournal(*pro, event.NilBus, fsys, "/mem/logbook.qfb")
	if err != nil {
		t.Fatal(err)
	}
	dsc := NewDscache(ctx, fsys, event.NilBus, "", "dscache.qfb")

	_, _, err = dsrefspec.GenerateExampleOplog(ctx, book, localDsName, "/ipfs/QmLocalExample")
	if err != nil {
		t.Fatal(err)
	}

	ref := dsref.Ref{
		Username: localUsername,
		Name:     localDsName,
	}
	if err := dsrefspec.ConsistentResolvers(t, ref, dsc, book); err != nil {
		t.Errorf("creating a dataset must update dscache")
		t.Errorf("inconsistent resolution between dscache & logbook:\n%s", err)
	}

	foreignUsername := "ref_consistency_foreign_user"
	foreignDsName := "example"
	foreignBook := dsrefspec.ForeignLogbook(t, foreignUsername)
	_, log, err := dsrefspec.GenerateExampleOplog(ctx, foreignBook, foreignDsName, "/ipfs/QmEXammPlle")
	if err != nil {
		t.Fatal(err)
	}

	if err := book.MergeLog(ctx, foreignBook.Owner().PubKey, log); err != nil {
		t.Fatal(err)
	}

	ref = dsref.Ref{
		Username: localUsername,
		Name:     localDsName,
	}
	if err := dsrefspec.ConsistentResolvers(t, ref, dsc, book); err != nil {
		t.Errorf("merging a foreign dataset must update dscache")
		t.Errorf("inconsistent resolution between dscache & logbook:\n%s", err)
	}
}
