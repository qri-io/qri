package dsref

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
)

// ErrNoHistory indicates a resolved reference has no HEAD path
var ErrNoHistory = fmt.Errorf("no history")

// ErrRefNotResolved is the error when LoadResolved is called on a non-resolved ref
var ErrRefNotResolved = fmt.Errorf("ref has not been resolved, cannot load")

// Loader loads datasets
type Loader interface {
	// LoadDataset will parse the ref string, resolve it, and load the dataset
	// from whatever store contains it
	LoadDataset(ctx context.Context, refstr string) (*dataset.Dataset, error)
	// LoadResolved loads a dataset that has already been resolved by finding it
	// at the given location. Will fail if the Ref has no Path with ErrRefNotResolved
	LoadResolved(ctx context.Context, ref Ref, location string) (*dataset.Dataset, error)
}
