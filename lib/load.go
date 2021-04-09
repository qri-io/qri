package lib

import (
	"context"
	"errors"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	qerr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/fsi"
)

type datasetLoader struct {
	inst   *Instance
	source string
}

// LoadDataset fetches, dereferences and opens a dataset from a reference
// implements the dsfs.Loader interface
// this function expects the passed in reference is fully resolved
func (d *datasetLoader) LoadResolved(ctx context.Context, ref dsref.Ref, location string) (*dataset.Dataset, error) {
	if ref.Path == "" {
		return nil, dsref.ErrRefNotResolved
	}
	if d == nil {
		return nil, fmt.Errorf("no datasetLoader")
	}
	if d.inst == nil {
		return nil, fmt.Errorf("no instance")
	}
	if location == "" {
		return d.loadLocalDataset(ctx, ref)
	}

	msg := fmt.Sprintf("pulling %s from %s ...\n", ref.Human(), location)
	if d.inst.streams.Out != nil {
		d.inst.streams.Out.Write([]byte(msg))
	}

	// TODO (b5) - it'd be nice to us the returned dataset here, skipping the
	// loadLocalDataset call entirely. For that to work dsfs.LoadDataset &
	// inst.loadLocalDataset would have to behave exactly the same, and currently
	// they don't
	if _, err := d.inst.remoteClient.PullDataset(ctx, &ref, location); err != nil {
		return nil, err
	}

	return d.loadLocalDataset(ctx, ref)
}

func (d *datasetLoader) loadLocalDataset(ctx context.Context, ref dsref.Ref) (*dataset.Dataset, error) {
	var (
		ds  *dataset.Dataset
		err error
	)

	if fsi.IsFSIPath(ref.Path) {
		// Has an FSI Path, load from working directory
		if ds, err = fsi.ReadDir(fsi.FilesystemPathToLocal(ref.Path)); err != nil {
			return nil, err
		}
	} else {
		// Load from dsfs
		if ds, err = dsfs.LoadDataset(ctx, d.inst.qfs, ref.Path); err != nil {
			return nil, err
		}
	}
	// Set transient info on the returned dataset
	ds.Name = ref.Name
	ds.Peername = ref.Username

	if err = base.OpenDataset(ctx, d.inst.repo.Filesystem(), ds); err != nil {
		log.Debugf("Get dataset, base.OpenDataset failed, error: %s", err)
		return nil, err
	}

	return ds, nil
}

func (d *datasetLoader) LoadDataset(ctx context.Context, refstr string) (*dataset.Dataset, error) {
	resolver, err := d.inst.resolverForSource(d.source)
	if err != nil {
		return nil, err
	}
	username := d.inst.cfg.Profile.Peername

	ref, err := dsref.Parse(refstr)
	if err != nil {
		return nil, fmt.Errorf("%q is not a valid dataset reference: %w", refstr, err)
	}

	if username == "" && ref.Username == "me" {
		msg := fmt.Sprintf(`Can't use the "me" keyword to refer to a dataset in this context.
Replace "me" with your username for the reference:
%s`, refstr)
		return nil, qerr.New(fmt.Errorf("invalid contextual reference"), msg)
	} else if username != "" && ref.Username == "me" {
		ref.Username = username
	}

	location, err := resolver.ResolveRef(ctx, &ref)
	if err != nil {
		if errors.Is(err, dsref.ErrRefNotFound) {
			return nil, qerr.New(err, fmt.Sprintf("reference %q not found", refstr))
		}
		return nil, err
	}

	if ref.Path == "" {
		err = qerr.New(dsref.ErrNoHistory, fmt.Sprintf("can't load dataset %q, it has no saved versions", ref.Human()))
		return nil, err
	}

	return d.LoadResolved(ctx, ref, location)
}
