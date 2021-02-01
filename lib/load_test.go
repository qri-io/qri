package lib

import (
	"context"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/event"
)

func TestLoadDataset(t *testing.T) {
	ctx := context.Background()
	tr := newTestRunner(t)
	defer tr.Delete()

	fs := tr.Instance.Repo().Filesystem()

	if _, err := (*Instance)(nil).LoadDataset(tr.Ctx, dsref.Ref{}, ""); err == nil {
		t.Errorf("expected loadDataset on a nil instance to fail without panicing")
	}

	dsrefspec.AssertLoaderSpec(t, tr.Instance, func(ds *dataset.Dataset) (string, error) {
		return dsfs.CreateDataset(
			tr.Ctx,
			fs,
			fs.DefaultWriteFS(),
			event.NilBus,
			ds,
			nil,
			tr.Instance.repo.PrivateKey(ctx),
			dsfs.SaveSwitches{},
		)
	})
}
