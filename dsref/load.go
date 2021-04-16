package dsref

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
)

// ErrNoHistory indicates a resolved reference has no HEAD path
var ErrNoHistory = fmt.Errorf("no history")

// Loader loads datasets
type Loader interface {
	// LoadDataset will parse the ref string, resolve it, and load the dataset
	// from whatever store contains it
	LoadDataset(ctx context.Context, refstr string) (*dataset.Dataset, error)
}
