package dsref

import (
	"context"

	"github.com/qri-io/dataset"
)

// Loader loads and opens a dataset. The only useful implementation of the
// loader interface is in github.com/qri-io/qri/lib.
// TODO(b5) - This interface is a work-in-progress
type Loader interface {
	LoadDataset(ctx context.Context, ref Ref, source string) (*dataset.Dataset, error)
}

// ParseResolveLoad is a function that combines dataset reference parsing,
// reference resolution, and loading in one function, turning a reference string
// into a dataset pointer
type ParseResolveLoad func(ctx context.Context, refStr string) (*dataset.Dataset, error)
