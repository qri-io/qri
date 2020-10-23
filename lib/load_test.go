package lib

import (
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
)

func TestLoadDataset(t *testing.T) {
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
			ds,
			nil,
			tr.Instance.repo.PrivateKey(),
			dsfs.SaveSwitches{},
		)
	})
}
