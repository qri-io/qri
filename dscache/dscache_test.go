package dscache

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qri-io/qfs/localfs"
	testPeers "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
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
	fs := localfs.NewFS()

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
