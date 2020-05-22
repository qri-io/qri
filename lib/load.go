package lib

import (
	"context"
	"fmt"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/fsi"
	reporef "github.com/qri-io/qri/repo/ref"
)

// LoadDataset fetches, derefences and opens a dataset from a reference
// implements the dsfs.Loader interface
func (inst *Instance) LoadDataset(ctx context.Context, ref dsref.Ref, source string) (*dataset.Dataset, error) {
	if source == "" {
		return inst.loadLocalDataset(ctx, ref)
	}

	// TODO(b5) - for now we're assuming any non-local source must fetch from the registry
	if inst.cfg.Registry == nil {
		return nil, fmt.Errorf("can't fetch remote dataset %q without a configured registry", ref)
	} else if inst.cfg.Registry.Location == "" {
		return nil, fmt.Errorf("can't fetch remote dataset %q without a configured registry", ref)
	}

	source = inst.cfg.Registry.Location

	msg := fmt.Sprintf("pulling dataset from registry: %s ...\n", ref)
	inst.streams.Out.Write([]byte(msg))

	if err := inst.remoteClient.CloneLogs(ctx, ref, source); err != nil {
		return nil, err
	}

	rref := reporef.RefFromDsref(ref)
	if err := inst.remoteClient.AddDataset(ctx, &rref, source); err != nil {
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
