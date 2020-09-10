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
	qerr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/fsi"
)

// LoadDataset fetches, dereferences and opens a dataset from a reference
// implements the dsfs.Loader interface
// this function expects the passed in reference is fully resolved
func (inst *Instance) LoadDataset(ctx context.Context, ref dsref.Ref, source string) (*dataset.Dataset, error) {
	if inst == nil {
		return nil, fmt.Errorf("no instance")
	}
	if source == "" {
		return inst.loadLocalDataset(ctx, ref)
	}

	// empty source assumes the registry
	// TODO (b5) - not sure we should even allow an empty source if it's expected
	// the ref is already resolved. The only case I can think of is a user-provided,
	// fully-resolved reference. Spec on Loading needs work
	if source == "" {
		if inst.cfg.Registry == nil {
			return nil, fmt.Errorf("can't fetch remote dataset %q without a configured registry", ref)
		} else if inst.cfg.Registry.Location == "" {
			return nil, fmt.Errorf("can't fetch remote dataset %q without a configured registry", ref)
		}
		source = inst.cfg.Registry.Location
	}

	msg := fmt.Sprintf("pulling %s from %s ...\n", ref.Human(), source)
	inst.streams.Out.Write([]byte(msg))

	// TODO (b5) - it'd be nice to us the returned dataset here, skipping the
	// loadLocalDataset call entirely. For that to work dsfs.LoadDataset &
	// inst.loadLocalDataset would have to behave exactly the same, and currently
	// they don't
	if _, err := inst.remoteClient.PullDataset(ctx, &ref, source); err != nil {
		return nil, err
	}

	return inst.loadLocalDataset(ctx, ref)
}

func (inst *Instance) loadLocalDataset(ctx context.Context, ref dsref.Ref) (*dataset.Dataset, error) {
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
		if ds, err = dsfs.LoadDataset(ctx, inst.qfs.DefaultWriteFS(), ref.Path); err != nil {
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

// NewParseResolveLoadFunc generates a dsref.ParseResolveLoad function from an
// instance
func (inst *Instance) NewParseResolveLoadFunc(remote string) (dsref.ParseResolveLoad, error) {
	resolver, err := inst.resolverForMode(remote)
	if err != nil {
		return nil, err
	}
	return NewParseResolveLoadFunc(inst.cfg.Profile.Peername, resolver, inst), nil
}

// NewParseResolveLoadFunc composes a username, resolver, and loader into a
// higher-order function that converts strings to full datasets
// pass the empty string as a username to disable the "me" keyword in references
func NewParseResolveLoadFunc(username string, resolver dsref.Resolver, loader dsref.Loader) dsref.ParseResolveLoad {
	return func(ctx context.Context, refStr string) (*dataset.Dataset, error) {
		ref, err := dsref.Parse(refStr)
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
			if errors.Is(err, dsref.ErrRefNotFound) {
				return nil, qerr.New(err, fmt.Sprintf("reference %q not found", refStr))
			}
			return nil, err
		}

		if ref.Path == "" {
			err = qerr.New(dsref.ErrNoHistory, fmt.Sprintf("can't load dataset %q, it has no saved versions", ref.Human()))
			return nil, err
		}

		return loader.LoadDataset(ctx, ref, source)
	}
}
