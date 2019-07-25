package lib

import (
	// "context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qri-io/deepdiff"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// Delta is an alias for deepdiff.Delta, abstracting the deepdiff implementation
// away from packages that depend on lib
type Delta = deepdiff.Delta

// DiffStat is an alias for deepdiff.Stat, abstracting the deepdiff implementation
// away from packages that depend on lib
type DiffStat = deepdiff.Stats

// DiffParams defines parameters for diffing two datasets with Diff
type DiffParams struct {
	LeftPath, RightPath string

	// Format       string
	// FormatConfig dataset.FormatConfig

	Selector string

	Limit, Offset int
	All           bool
}

// DiffResponse is the result of a call to diff
type DiffResponse struct {
	Stat *DiffStat   `json:"stat,omitempty"`
	Diff []*Delta    `json:"diff,omitempty"`
	A    interface{} `json:"b,omitempty"`
	B    interface{} `json:"a,omitempty"`
}

// Diff computes the diff of two datasets
func (r *DatasetRequests) Diff(p *DiffParams, res *DiffResponse) (err error) {
	// absolutize any local paths before a possible trip over RPC to another local process
	if !repo.IsRefString(p.LeftPath) {
		if err = qfs.AbsPath(&p.LeftPath); err != nil {
			return
		}
	}
	if !repo.IsRefString(p.RightPath) {
		if err = qfs.AbsPath(&p.RightPath); err != nil {
			return
		}
	}

	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Diff", p, res)
	}

	if err = completeDiffRefs(r.node, &p.LeftPath, &p.RightPath); err != nil {
		return
	}

	var leftData, rightData interface{}
	if leftData, err = r.loadDiffData(p.LeftPath, p.Selector); err != nil {
		return
	}
	if rightData, err = r.loadDiffData(p.RightPath, p.Selector); err != nil {
		return
	}

	_res := DiffResponse{
		Stat: &deepdiff.Stats{},
		A:    leftData,
		B:    rightData,
	}

	if _res.Diff, err = deepdiff.Diff(leftData, rightData, deepdiff.OptionSetStats(_res.Stat)); err != nil {
		return
	}

	*res = _res
	return
}

func completeDiffRefs(node *p2p.QriNode, left, right *string) (err error) {
	// fail if neither argument is given
	if *left == "" && *right == "" {
		return repo.ErrEmptyRef
	}
	// fill in left side from previous path if left isn't set & right is a ref string with history
	if *right != "" && *left == "" && repo.IsRefString(*right) {
		if *right == "" {
			return repo.ErrEmptyRef
		}
		ref := *right

		lr := NewLogRequests(node, nil)
		var res []repo.DatasetRef
		err = lr.Log(&LogParams{
			Ref: ref,
			ListParams: ListParams{
				Limit:  10,
				Offset: 0,
			},
		}, &res)
		if err != nil {
			return
		}

		if len(res) == 0 {
			// NOTE: This shouldn't be possible.
			return fmt.Errorf("dataset has no versions, nothing to diff against")
		} else if len(res) == 1 {
			return fmt.Errorf("dataset has only one version, nothing to diff against")
		} else {
			*left = res[1].String()
		}
	}

	return
}

// TODO (b5): this is a temporary hack, I'd like to eventually merge this with a
// bunch of other code, generalizing the types of data qri can work on
func (r *DatasetRequests) loadDiffData(path, selector string) (data interface{}, err error) {
	if repo.IsRefString(path) {
		getp := &GetParams{
			Path:     path,
			Format:   "json",
			Selector: selector,
			All:      true,
		}
		res := &GetResult{}
		if err = r.Get(getp, res); err != nil {
			return
		}
		err = json.Unmarshal(res.Bytes, &data)
		return
	}
	file, err := r.node.Repo.Filesystem().Get(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(file.FileName()))
	switch ext {
	case ".json":
		err = json.NewDecoder(file).Decode(&data)
		// err = codec.NewDecoder(file, &codec.JsonHandle{}).Decode(&data)
	case ".csv":
		data, err = allCSVRows(file)
	case ".cbor":
		err = fmt.Errorf("cbor is not yet supported")
		// err = codec.NewDecoder(file, &codec.CborHandle{}).Decode(&data)
	default:
		err = fmt.Errorf("unrecognized file extension: %s", ext)
	}
	return
}

func allCSVRows(file qfs.File) (recs []interface{}, err error) {
	rdr := csv.NewReader(file)

	for {
		var rec []string
		if rec, err = rdr.Read(); err != nil {
			if err.Error() == "EOF" {
				err = nil
				break
			}
			return nil, err
		}
		recs = append(recs, rec)
	}
	return recs, nil
}

// MergeDiffs merges a list of DiffResponses into another, adding component names to each path
func (r *DatasetRequests) MergeDiffs(merged *DiffResponse, inputs []DiffResponse, comps []string) (err error) {
	merged.Stat = &DiffStat{}
	for i, inp := range inputs {
		merged.Stat.Left += inp.Stat.Left
		merged.Stat.Right += inp.Stat.Right
		merged.Stat.LeftWeight += inp.Stat.LeftWeight
		merged.Stat.RightWeight += inp.Stat.RightWeight
		merged.Stat.Inserts += inp.Stat.Inserts
		merged.Stat.Updates += inp.Stat.Updates
		merged.Stat.Deletes += inp.Stat.Deletes
		merged.Stat.Moves += inp.Stat.Moves
		for j, d := range inp.Diff {
			inp.Diff[j].Path = comps[i] + "/" + d.Path
		}
		merged.Diff = append(merged.Diff, inp.Diff...)
	}
	return nil
}
