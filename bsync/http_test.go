package bsync

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qri-io/qri/manifest"

	files "gx/ipfs/QmZMWMvWMVKCbHetJ4RgndbuEF1io2UpUxwQwtNjtYPzSC/go-ipfs-files"
)

func TestSyncHTTP(t *testing.T) {
	ctx := context.Background()
	_, a, err := makeAPI(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, b, err := makeAPI(ctx)
	if err != nil {
		t.Fatal(err)
	}

	f := files.NewReaderFile("oh_hey", "oh_hey", ioutil.NopCloser(strings.NewReader("y"+strings.Repeat("o", 35000))), nil)
	path, err := a.Unixfs().Add(ctx, f)
	if err != nil {
		t.Fatal(err)
	}

	aGetter := &manifest.NodeGetter{Dag: a.Dag()}
	mfst, err := manifest.NewManifest(ctx, aGetter, path.Cid())
	if err != nil {
		t.Fatal(err)
	}

	bGetter := &manifest.NodeGetter{Dag: b.Dag()}
	rs := NewReceivers(ctx, bGetter, b.Block())
	s := httptest.NewServer(http.HandlerFunc(rs.Handler))
	defer s.Close()

	rem := &HTTPRemote{URL: s.URL}

	send, err := NewSend(ctx, aGetter, mfst, rem)
	if err != nil {
		t.Fatal(err)
	}

	if err := send.Do(); err != nil {
		t.Error(err)
	}

	// b should now be able to generate a manifest
	_, err = manifest.NewManifest(ctx, bGetter, path.Cid())
	if err != nil {
		t.Error(err)
	}
}
