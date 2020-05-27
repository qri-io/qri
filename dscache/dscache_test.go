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
	testPeers "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/repo/profile"
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

	peerInfo := testPeers.GetTestPeerInfo(0)
	peername := "test_user"

	// Construct a dscache, will not save without a filename
	builder := NewBuilder()
	builder.AddUser(peername, profile.IDFromPeerID(peerInfo.PeerID).String())
	builder.AddDsVersionInfo(dsref.VersionInfo{InitID: "abcd1"})
	builder.AddDsVersionInfo(dsref.VersionInfo{InitID: "efgh2"})
	constructed := builder.Build()

	// A dscache that will save when it is assigned
	dscacheFile := filepath.Join(tmpdir, "dscache.qfb")
	saveable := NewDscache(ctx, fs, nil, peername, dscacheFile)
	saveable.Assign(constructed)

	// Load the dscache from its serialized file, verify it has correct data
	loadable := NewDscache(ctx, fs, nil, peername, dscacheFile)
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
	dsc := NewDscache(ctx, fs, nil, "test_resolve_ref_user", path)

	dsrefspec.AssertResolverSpec(t, dsc, func(r dsref.Ref, _ identity.Author, _ *oplog.Log) error {
		builder := NewBuilder()
		peerInfo := testPeers.GetTestPeerInfo(0)
		builder.AddUser(r.Username, peerInfo.EncodedPeerID)
		builder.AddDsVersionInfo(dsref.VersionInfo{
			Username:  r.Username,
			InitID:    r.InitID,
			Path:      r.Path,
			ProfileID: peerInfo.EncodedPeerID,
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
	book, err := logbook.NewJournal(testPeers.GetTestPeerInfo(0).PrivKey, localUsername, fsys, "/mem/logbook")
	if err != nil {
		t.Fatal(err)
	}
	dsc := NewDscache(ctx, fsys, nil, "", "dscache.qfb")

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

	if err := book.MergeLog(ctx, foreignBook.Author(), log); err != nil {
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
