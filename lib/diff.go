package lib

import (
	"encoding/json"

	"github.com/qri-io/difff"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// Delta is an alias for diff.Delta, abstracting the difff implementation
// away from packages that depend on lib
type Delta = difff.Delta

// DiffParams defines parameters for diffing two datasets with Diff
type DiffParams struct {
	LeftPath, RightPath string

	// Format       string
	// FormatConfig dataset.FormatConfig

	Selector string

	Concise       bool
	Limit, Offset int
	All           bool
}

// Diff computes the diff of two datasets
func (r *DatasetRequests) Diff(p *DiffParams, changes *[]*Delta) (err error) {

	left, right, err := diffRefs(r.node, p.LeftPath, p.RightPath)
	if err != nil {
		return
	}

	rp := &GetParams{
		Path:     right.String(),
		Format:   "json",
		Selector: p.Selector,
		Concise:  p.Concise,
		All:      true,
	}
	rightRes := &GetResult{}
	if err = r.Get(rp, rightRes); err != nil {
		return err
	}
	var rightData interface{}
	if err = json.Unmarshal(rightRes.Bytes, &rightData); err != nil {
		return err
	}

	lp := &GetParams{
		Path:     left.String(),
		Format:   "json",
		Selector: p.Selector,
		Concise:  p.Concise,
		All:      true,
	}
	leftRes := &GetResult{}
	if err = r.Get(lp, leftRes); err != nil {
		return err
	}
	var leftData interface{}
	if err = json.Unmarshal(leftRes.Bytes, &leftData); err != nil {
		return err
	}

	// *diffs, err = actions.DiffDatasets(r.node, p.Left, p.Right, p.DiffAll, p.DiffComponents)
	*changes = difff.Diff(leftData, rightData)
	return
}

func diffRefs(node *p2p.QriNode, leftPath, rightPath string) (left, right repo.DatasetRef, err error) {
	left, err = repo.ParseDatasetRef(leftPath)
	if err != nil && err != repo.ErrEmptyRef {
		return
	}
	right, err = repo.ParseDatasetRef(rightPath)
	if err != nil && err != repo.ErrEmptyRef {
		return
	}

	refs := []repo.DatasetRef{}
	// Handle `qri use` to get the current default dataset.
	if err = DefaultSelectedRefs(node.Repo, &refs); err != nil {
		return
	}

	if len(refs) >= 2 && left.IsEmpty() && right.IsEmpty() {
		left = refs[0]
		right = refs[1]
	} else if right.IsEmpty() && len(refs) == 1 {
		// accommodate using a single ref
		left = refs[1]
	}

	// fill in left side from previous path if left isn't set
	if !right.IsEmpty() && left.IsEmpty() {
		lr := NewLogRequests(node, nil)
		var res []repo.DatasetRef
		err = lr.Log(&LogParams{
			ListParams: ListParams{
				Limit:  10,
				Offset: 0,
			},
			Ref: right,
		}, &res)
		if err != nil {
			return
		}
		if len(res) > 0 {
			left = res[1]
		}
	}

	return
}
