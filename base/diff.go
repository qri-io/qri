package base

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset/dsdiff"
	"github.com/qri-io/qri/repo"
)

// DiffDatasets calculates the difference between two dataset references
func DiffDatasets(ctx context.Context, r repo.Repo, leftRef, rightRef repo.DatasetRef, all bool, components map[string]bool) (diffs map[string]*dsdiff.SubDiff, err error) {
	if leftRef.IsEmpty() || rightRef.IsEmpty() {
		// TODO - make new error
		err = fmt.Errorf("please provide two dataset references to compare")
		return
	}

	if err = ReadDataset(ctx, r, &leftRef); err != nil {
		return nil, err
	}
	dsLeft := leftRef.Dataset

	if err = ReadDataset(ctx, r, &rightRef); err != nil {
		return nil, err
	}
	dsRight := rightRef.Dataset

	diffs = make(map[string]*dsdiff.SubDiff)
	if all {
		if diffs, err = dsdiff.DiffDatasets(dsLeft, dsRight, nil); err != nil {
			log.Debug(err.Error())
			err = fmt.Errorf("error diffing datasets: %s", err.Error())
			return
		}
		// TODO: remove this temporary hack
		if diffs["data"] == nil || len(diffs["data"].Deltas()) == 0 {
			// dereference data paths
			// marshal json to []byte
			// call `dsdiff.DiffJSON(a, b)`
		}
	} else {
		for k, v := range components {
			if v {
				switch k {
				case "structure":
					if dsLeft.Structure != nil && dsRight.Structure != nil {
						structureDiffs, e := dsdiff.DiffStructure(dsLeft.Structure, dsRight.Structure)
						if e != nil {
							err = fmt.Errorf("error diffing %s: %s", k, e.Error())
							return
						}
						diffs[k] = structureDiffs
					}
				case "data":
					//TODO
					if dsLeft.BodyPath != "" && dsRight.BodyPath != "" {
						dataDiffs, e := dsdiff.DiffData(dsLeft, dsRight)
						if e != nil {
							err = fmt.Errorf("error diffing %s: %s", k, e.Error())
							return
						}
						diffs[k] = dataDiffs
					}
				case "transform":
					if dsLeft.Transform != nil && dsRight.Transform != nil {
						transformDiffs, e := dsdiff.DiffTransform(dsLeft.Transform, dsRight.Transform)
						if e != nil {
							err = fmt.Errorf("error diffing %s: %s", k, e.Error())
							return
						}
						diffs[k] = transformDiffs
					}
				case "meta":
					if dsLeft.Meta != nil && dsRight.Meta != nil {
						metaDiffs, e := dsdiff.DiffMeta(dsLeft.Meta, dsRight.Meta)
						if e != nil {
							err = fmt.Errorf("error diffing %s: %s", k, e.Error())
							return
						}
						diffs[k] = metaDiffs
					}
				case "viz":
					if dsLeft.Viz != nil && dsRight.Viz != nil {
						vizDiffs, e := dsdiff.DiffViz(dsLeft.Viz, dsRight.Viz)
						if e != nil {
							err = fmt.Errorf("error diffing %s: %s", k, e.Error())
							return
						}
						diffs[k] = vizDiffs
					}
				}
			}
		}
	}

	return
}
