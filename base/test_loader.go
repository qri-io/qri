package base

import (
	"context"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
)

type loader struct {
	fs       qfs.Filesystem
	resolver dsref.Resolver
}

// NewTestDatasetLoader constructs a loader that is useful for tests
// since they only need a simplified version of resolution and loading
func NewTestDatasetLoader(fs qfs.Filesystem, resolver dsref.Resolver) dsref.Loader {
	return loader{
		fs:       fs,
		resolver: resolver,
	}
}

// LoadDataset loads a dataset
func (l loader) LoadDataset(ctx context.Context, refstr string) (*dataset.Dataset, error) {
	ref, err := dsref.Parse(refstr)
	if err != nil {
		return nil, err
	}

	_, err = l.resolver.ResolveRef(ctx, &ref)
	if err != nil {
		return nil, err
	}

	ds, err := dsfs.LoadDataset(ctx, l.fs, ref.Path)
	if err != nil {
		return nil, err
	}

	err = OpenDataset(ctx, l.fs, ds)
	return ds, err
}
