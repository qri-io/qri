package remote

import (
	"context"
	"testing"

	"github.com/qri-io/qfs"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
)

func TestMockClient(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	kd := testkeys.GetKeyData(5)
	fs, err := qfs.NewMemFilesystem(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	book, err := logbook.NewJournal(kd.PrivKey, "example_uesr", event.NilBus, fs, "logbook.qfb")
	if err != nil {
		t.Fatal(err)
	}

	c, err := NewMockClient(tr.Ctx, tr.NodeA, book)
	if err != nil {
		t.Fatal(err)
	}

	ref := dsref.MustParse("this/should_get_made_on_the_fly")
	if _, err := c.PullDataset(tr.Ctx, &ref, ""); err != nil {
		t.Error(err)
	}

	resolve := dsref.MustParse("wut/create_me")
	if _, err := c.NewRemoteRefResolver("").ResolveRef(tr.Ctx, &resolve); err != nil {
		t.Error(err)
	}

	ref = dsref.MustParse("wut/create_me")
	if _, err = c.PullDataset(tr.Ctx, &ref, ""); err != nil {
		t.Error(err)
	}
}
