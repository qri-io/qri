package actions

import (
	"fmt"

	"github.com/qri-io/dsdiff"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// DiffDatasets calculates the difference between two dataset references
func DiffDatasets(node *p2p.QriNode, leftRef, rightRef repo.DatasetRef, all bool, components map[string]bool) (diffs map[string]*dsdiff.SubDiff, err error) {
	if leftRef.IsEmpty() || rightRef.IsEmpty() {
		// TODO - make new error
		err = fmt.Errorf("please provide two dataset references to compare")
		return
	}

	if err = DatasetHead(node, &leftRef); err != nil {
		return
	}
	dsLeft, e := leftRef.DecodeDataset()
	if e != nil {
		err = e
		return
	}

	if err = DatasetHead(node, &rightRef); err != nil {
		return
	}
	dsRight, e := rightRef.DecodeDataset()
	if e != nil {
		err = e
		return
	}

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
	// Hack to examine data
	// if all || components["data"] == true {
	// 	if dsLeft.Structure.Checksum == dsRight.Structure.Checksum {
	// 		return
	// 	}
	// 	params0 := &LookupParams{
	// 		Format: dataset.JSONDataFormat,
	// 		Path:   dsLeft.Path().String(),
	// 	}
	// 	params1 := &LookupParams{
	// 		Format: dataset.JSONDataFormat,
	// 		Path:   dsRight.Path().String(),
	// 	}
	// 	result0 := &LookupResult{}
	// 	result1 := &LookupResult{}
	// 	err := r.LookupBody(params0, result0)
	// 	if err != nil {
	// 		log.Debug(err.Error())
	// 		return fmt.Errorf("error getting structured data: %s", err.Error())
	// 	}
	// 	err = r.LookupBody(params1, result1)
	// 	if err != nil {
	// 		log.Debug(err.Error())
	// 		return fmt.Errorf("error getting structured data: %s", err.Error())
	// 	}

	// 	m0 := &map[string]json.RawMessage{"data": result0.Data}
	// 	m1 := &map[string]json.RawMessage{"data": result1.Data}
	// 	dataBytes0, err := json.Marshal(m0)
	// 	if err != nil {
	// 		log.Debug(err.Error())
	// 		return fmt.Errorf("error marshaling json: %s", err.Error())
	// 	}
	// 	dataBytes1, err := json.Marshal(m1)
	// 	if err != nil {
	// 		log.Debug(err.Error())
	// 		return fmt.Errorf("error marshaling json: %s", err.Error())
	// 	}
	// 	dataDiffs, err := dsdiff.DiffJSON(dataBytes0, dataBytes1, "data")
	// 	if err != nil {
	// 		log.Debug(err.Error())
	// 		return fmt.Errorf("error comparing structured data: %s", err.Error())
	// 	}
	// 	diffs["data"] = dataDiffs
	// }

	return
}
