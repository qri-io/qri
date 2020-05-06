package lib

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/fsi"
)

// ParseAndResolveRef combines reference parsing and resolution
func (inst *Instance) ParseAndResolveRef(ctx context.Context, refStr, source string) (dsref.Ref, string, error) {
	ref, err := dsref.Parse(refStr)

	// bad case references are allowed-but-warned for backwards compatibility
	if errors.Is(err, dsref.ErrBadCaseName) {
		log.Error(dsref.ErrBadCaseShouldRename)
		err = nil
	} else if err != nil {
		return ref, "", fmt.Errorf("%q is not a valid dataset reference: %w", refStr, err)
	}

	resolvedSource, err := inst.ResolveReference(ctx, &ref, source)
	return ref, resolvedSource, err
}

// ResolveReference finds the identifier & HEAD path for a dataset reference.
// the source parameter determines which subsystems of Qri to use when
func (inst *Instance) ResolveReference(ctx context.Context, ref *dsref.Ref, source string) (string, error) {
	if inst == nil {
		return "", dsref.ErrNotFound
	}

	// Handle the "me" convenience shortcut
	if ref.Username == "me" {
		// TODO (b5) - this should be reading from a better place, and erroring if
		// a canonical profile cannot be found for whatever reason
		ref.Username = inst.cfg.Profile.Peername
	}

	resolvers, err := inst.resolveSources(source)
	if err != nil {
		return "", err
	}

	for _, resolver := range resolvers {
		resolvedSource, err := resolver.ResolveRef(ctx, ref)
		if err != nil {
			if errors.Is(err, dsref.ErrNotFound) {
				continue
			} else {
				return "", err
			}
		}

		return resolvedSource, nil
	}

	return "", dsref.ErrNotFound
}

func (inst *Instance) resolveSources(source string) ([]dsref.Resolver, error) {
	switch source {
	case "":
		return []dsref.Resolver{
			inst.dscache,
			inst.repo,
			dsref.ParallelResolver(
				inst.logbook,
				// inst.registry,
				// inst.node,
			),
		}, nil
	case "local":
		return []dsref.Resolver{
			inst.dscache,
			inst.repo,
			inst.logbook,
		}, nil
	case "network":
		return nil, fmt.Errorf("network resolution not finished")
		// return dsref.ParallelResolver(
		// 	inst.registry,
		// 	// inst.node,
		// ), nil
	case "registry":
		return nil, fmt.Errorf("network resolution not finished")
		// return []dsref.Resolver{inst.registry}, nil
	case "p2p":
		return nil, fmt.Errorf("p2p network cannot be used to resolve references")
	}

	// TODO (b5) - sources could be one of:
	// * configured remote name
	// * peername
	// * peer multiaddress
	return nil, fmt.Errorf("unknown source: %q", source)
}

// loadDataset fetches, derefences and opens a dataset from a reference
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

	if err = base.OpenDataset(ctx, inst.repo.Filesystem(), ds); err != nil {
		log.Debugf("Get dataset, base.OpenDataset failed, error: %s", err)
		return nil, err
	}

	return ds, nil
}
