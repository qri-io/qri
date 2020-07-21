package remote

import (
	"context"
	"testing"

	"github.com/qri-io/qfs"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
)

func TestMockClient(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	pi := cfgtest.GetTestPeerInfo(5)
	fs, err := qfs.NewMemFilesystem(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	book, err := logbook.NewJournal(pi.PrivKey, "example_uesr", event.NilBus, fs, "logbook.qfb")
	if err != nil {
		t.Fatal(err)
	}

	c, err := NewMockClient(tr.Ctx, tr.NodeA, book)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.CloneLogs(tr.Ctx, dsref.MustParse("this/should_get_made_on_the_fly"), ""); err != nil {
		t.Error(err)
	}

	resolve := dsref.MustParse("wut/create_me")
	if _, err := c.NewRemoteRefResolver("").ResolveRef(tr.Ctx, &resolve); err != nil {
		t.Error(err)
	}

	if err = c.CloneLogs(tr.Ctx, dsref.MustParse("wut/create_me"), ""); err != nil {
		t.Error(err)
	}
}
