package base

import (
	"context"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
)

// LoadRevs grabs a component of a dataset that exists <n>th generation ancestor
// of the referenced version, where presence of a component in a previous snapshot constitutes ancestry
func LoadRevs(ctx context.Context, r repo.Repo, ref repo.DatasetRef, revs []*dsref.Rev) (res *dataset.Dataset, err error) {
	var ds *dataset.Dataset
	res = &dataset.Dataset{}
	for {
		if ds, err = dsfs.LoadDataset(ctx, r.Store(), ref.Path); err != nil {
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
