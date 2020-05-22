package dsref

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	qerr "github.com/qri-io/qri/errors"
)

// Loader loads and opens a dataset. The only useful implementation of the
// loader interface is in github.com/qri-io/qri/lib.
// TODO(b5) - This interface is a work-in-progress
type Loader interface {
	LoadDataset(ctx context.Context, ref Ref, source string) (*dataset.Dataset, error)
}

// ParseResolveLoad is the function type returned by NewParseResolveLoader
type ParseResolveLoad func(ctx context.Context, refStr string) (*dataset.Dataset, error)

// NewParseResolveLoadFunc composes a username, resolver, and loader into a
// higher-order function that converts strings to full datasets
// pass the empty string as a username to disable the "me" keyword in references
func NewParseResolveLoadFunc(username string, resolver Resolver, loader Loader) ParseResolveLoad {
	return func(ctx context.Context, refStr string) (*dataset.Dataset, error) {
		ref, err := Parse(refStr)
		if err != nil {
			return nil, err
		}

		if username == "" && ref.Username == "me" {
			msg := fmt.Sprintf(`Can't use the "me" keyword to refer to a dataset in this context.
Replace "me" with your username for the reference:
%s`, refStr)
			return nil, qerr.New(fmt.Errorf("invalid contextual reference"), msg)
		} else if username != "" && ref.Username == "me" {
			ref.Username = username
		}

		source, err := resolver.ResolveRef(ctx, &ref)
		if err != nil {
			return nil, err
		}

		return loader.LoadDataset(ctx, ref, source)
	}
}
