package base

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
)

// Recall loads revisions of a dataset from history of a resolved dataset
// reference
func Recall(ctx context.Context, store cafs.Filestore, ref dsref.Ref, revStr string) (*dataset.Dataset, error) {
	if revStr == "" {
		return &dataset.Dataset{}, nil
	}
	if ref.Path == "" {
		return nil, fmt.Errorf("can only recall from a resolved reference with a path value")
	}

	revs, err := dsref.ParseRevs(revStr)
	if err != nil {
		return nil, err
	}

	res, err := LoadRevs(ctx, store, ref, revs)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// LoadRevs grabs a component of a dataset that exists <n>th generation ancestor
// of the referenced version, where presence of a component in a previous snapshot constitutes ancestry
func LoadRevs(ctx context.Context, store cafs.Filestore, ref dsref.Ref, revs []*dsref.Rev) (res *dataset.Dataset, err error) {
	var ds *dataset.Dataset
	res = &dataset.Dataset{}
	for {
		if ds, err = dsfs.LoadDataset(ctx, store, ref.Path); err != nil {
			return
		}

		done := true
		for _, rev := range revs {
			if !sel(rev, ds, res) && done {
				done = false
			}
		}
		if done {
			break
		}

		ref.Path = ds.PreviousPath
		if ref.Path == "" {
			break
		}
	}
	return res, nil
}

func sel(r *dsref.Rev, ds, res *dataset.Dataset) bool {
	switch r.Field {
	case "ds":
		r.Gen--
		if r.Gen == 0 {
			res = ds
		}
	case "bd":
		if ds.BodyPath != "" {
			r.Gen--
			if r.Gen == 0 {
				res.BodyPath = ds.BodyPath
			}
		}
	case "md":
		if ds.Meta != nil {
			r.Gen--
			if r.Gen == 0 {
				res.Meta = ds.Meta
			}
		}
	case "tf":
		if ds.Transform != nil {
			r.Gen--
			if r.Gen == 0 {
				res.Transform = ds.Transform
			}
		}
	case "cm":
		if ds.Commit != nil {
			r.Gen--
			if r.Gen == 0 {
				res.Commit = ds.Commit
			}
		}
	case "vz":
		if ds.Viz != nil {
			r.Gen--
			if r.Gen == 0 {
				res.Viz = ds.Viz
			}
		}
	case "rm":
		if ds.Readme != nil {
			r.Gen--
			if r.Gen == 0 {
				res.Readme = ds.Readme
			}
		}
	case "st":
		if ds.Structure != nil {
			r.Gen--
			if r.Gen == 0 {
				res.Structure = ds.Structure
			}
		}
	}

	return r.Gen == 0
}

// Drop sets named components to nil from a revision string
func Drop(ds *dataset.Dataset, revStr string) error {
	if revStr == "" {
		return nil
	}
	log.Debugf("Drop revStr=%q", revStr)

	revs, err := dsref.ParseRevs(revStr)
	if err != nil {
		return err
	}

	for _, rev := range revs {
		if rev.Gen != 1 {
			return fmt.Errorf("cannot drop specific generations")
		}
		switch rev.Field {
		case "md":
			ds.Meta = nil
		case "vz":
			ds.Viz = nil
		case "tf":
			ds.Transform = nil
		case "st":
			ds.Structure = nil
		case "bd":
			ds.Body = nil
		case "rm":
			ds.Readme = nil
		default:
			return fmt.Errorf("cannot drop component: %q", rev.Field)
		}
	}

	return nil
}
