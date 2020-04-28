package lib

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/fsi"
)

// assert at compile time that instance is a RefResolver
var _ dsref.RefResolver = (*Instance)(nil)

// ParseAndResolveRef combines reference parsing and resolution
func (inst *Instance) ParseAndResolveRef(ctx context.Context, refStr string) (dsref.Ref, error) {
	ref, err := dsref.Parse(refStr)

	// bad case references are allowed-but-warned for backwards compatibility
	if errors.Is(err, dsref.ErrBadCaseName) {
		log.Error(dsref.ErrBadCaseShouldRename)
		err = nil
	} else if err != nil {
		return ref, fmt.Errorf("%q is not a valid dataset reference: %w", refStr, err)
	}

	err = inst.ResolveRef(ctx, &ref)
	return ref, err
}

// ResolveRef finds the identifier for a dataset reference
func (inst *Instance) ResolveRef(ctx context.Context, ref *dsref.Ref) error {
	if inst == nil {
		return dsref.ErrNotFound
	}

	resolvers := []dsref.RefResolver{
		// local resolution
		inst.dscache,
		inst.repo,
		inst.logbook,

		// network resolution
		// inst.registry,
		// inst.QriNode
	}

	for _, r := range resolvers {
		err := r.ResolveRef(ctx, ref)
		if err == nil {
			return nil
		} else if errors.Is(err, dsref.ErrNotFound) {
			continue
		}

		return err
	}

	return dsref.ErrNotFound
}

// TODO (b5) - this needs to move down into base, replacing base.LoadDataset with
// a version that can load paths with a /fsi prefix, but before that can happen
// base needs to be able to import FSI. Currently FSI imports base, and doesn't
// really need to. Most of the base package functions used by FSI should be in
// FSI, as they deal with filesystem interaction.
func (inst *Instance) loadDataset(ctx context.Context, ref dsref.Ref) (*dataset.Dataset, error) {
	var (
		ds  *dataset.Dataset
		err error
	)

	if strings.HasPrefix(ref.Path, "/fsi") {
		// Has an FSI Path, load from working directory
		if ds, err = fsi.ReadDir(strings.TrimPrefix(ref.Path, "/fsi")); err != nil {
			return nil, err
		}
	} else {
		// Load from dsfs
		if ds, err = dsfs.LoadDataset(ctx, inst.store, ref.Path); err != nil {
			return nil, err
		}
	}
	// Set transient info on the returned dataset
	ds.Name = ref.Name
	ds.Peername = ref.Username
	return ds, nil
}
