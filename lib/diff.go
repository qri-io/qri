package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/deepdiff"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/base/dsfs"
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
	// File path or reference to a dataset
	LeftPath, RightPath string

	// Which component or part of a dataset to compare
	Selector string

	// If not null, the working directory that the diff is using
	WorkingDir string
	// Whether to get the previous version of the left parameter
	IsLeftAsPrevious bool

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
	ctx := context.TODO()

	if !repo.IsRefString(p.LeftPath) && !repo.IsRefString(p.RightPath) {
		// Compare body files.
		leftComp := component.NewBodyComponent(p.LeftPath)
		leftData, err := leftComp.StructuredData()
		if err != nil {
			return err
		}

		rightComp := component.NewBodyComponent(p.RightPath)
		rightData, err := rightComp.StructuredData()
		if err != nil {
			return err
		}

		res.Stat = &deepdiff.Stats{}
		res.A = leftData
		res.B = rightData
		res.Diff, err = deepdiff.Diff(leftData, rightData, deepdiff.OptionSetStats(res.Stat))
		return err
	} else if !repo.IsRefString(p.LeftPath) || !repo.IsRefString(p.RightPath) {
		// Only one is a file path, other is a reference. Cannot compare.
		return fmt.Errorf("Cannot compare a dataset reference against a body file")
	}

	// Left side of diff
	ref, err := repo.ParseDatasetRef(p.LeftPath)
	if err != nil {
		return err
	}
	err = repo.CanonicalizeDatasetRef(r.inst.node.Repo, &ref)
	if err != nil {
		if err == repo.ErrNoHistory {
			return fmt.Errorf("dataset has no versions, nothing to diff against")
		}
		return err
	}
	ds, err := dsfs.LoadDataset(ctx, r.inst.node.Repo.Store(), ref.Path)
	if err != nil {
		return err
	}
	if p.IsLeftAsPrevious {
		prev := ds.PreviousPath
		if prev == "" {
			return fmt.Errorf("dataset has only one version, nothing to diff against")
		}
		ref.Path = prev
		ds, err = dsfs.LoadDataset(ctx, r.inst.node.Repo.Store(), ref.Path)
		if err != nil {
			return err
		}
	}
	leftComp := component.ConvertDatasetToComponents(ds, r.inst.node.Repo.Filesystem())

	// Right side of diff
	var rightComp component.Component
	if p.WorkingDir != "" {
		// Working directory, read dataset from the current files.
		rightComp, err = component.ListDirectoryComponents(p.WorkingDir)
		if err != nil {
			return err
		}
		err = component.ExpandListedComponents(rightComp, r.inst.node.Repo.Filesystem())
		if err != nil {
			return err
		}
		// TODO(dlong): Hack! This is what fills the value. StucturedData assumes this has been
		// called. Should cleanup component's API so that this isn't necessary.
		_, err = component.ToDataset(rightComp)
		if err != nil {
			return err
		}

	} else {
		ref, err := repo.ParseDatasetRef(p.RightPath)
		if err != nil {
			return err
		}
		err = repo.CanonicalizeDatasetRef(r.inst.node.Repo, &ref)
		if err != nil && err != repo.ErrNoHistory {
			return err
		}
		ds, err := dsfs.LoadDataset(ctx, r.inst.node.Repo.Store(), ref.Path)
		if err != nil {
			return err
		}
		rightComp = component.ConvertDatasetToComponents(ds, r.inst.node.Repo.Filesystem())
	}

	// If in an FSI linked working directory, drop derived values, since the user is not
	// expected to have those trasient values on their checked out files.
	if p.WorkingDir != "" {
		// TODO(dlong): RemoveSubcomponent removes the component from the map, but not from the
		// Value. That should be fixed so that component has a more sane API.
		leftComp.Base().RemoveSubcomponent("commit")
		leftComp.Base().RemoveSubcomponent("viz")
		leftComp.DropDerivedValues()
		rightComp.Base().RemoveSubcomponent("commit")
		rightComp.Base().RemoveSubcomponent("viz")
		rightComp.DropDerivedValues()

		// Also load the body file, and inline it.
		// TODO(dlong): This should be refactored into component so that it's easier to do.
		leftDsComp := leftComp.Base().GetSubcomponent("dataset")
		if leftDsComp != nil {
			dsComp, ok := leftDsComp.(*component.DatasetComponent)
			if ok {
				ds := dsComp.Value
				ds.Commit = nil
				ds.Viz = nil
				ds.Peername = ""
				ds.PreviousPath = ""
				bodyComp := leftComp.Base().GetSubcomponent("body")
				if bodyComp != nil {
					bodyComp.LoadAndFill(ds)
					ds.Body, err = bodyComp.StructuredData()
					if err != nil {
						return err
					}
					ds.BodyPath = ""
				}
			}
		}

		rightDsComp := rightComp.Base().GetSubcomponent("dataset")
		if rightDsComp != nil {
			dsComp, ok := rightDsComp.(*component.DatasetComponent)
			if ok {
				ds := dsComp.Value
				ds.Commit = nil
				ds.Viz = nil
				ds.Peername = ""
				ds.PreviousPath = ""
				bodyComp := rightComp.Base().GetSubcomponent("body")
				if bodyComp != nil {
					bodyComp.LoadAndFill(ds)
					ds.Body, err = bodyComp.StructuredData()
					if err != nil {
						return err
					}
					ds.BodyPath = ""
				}
			}
		}
	}

	selector := p.Selector
	if selector == "" {
		selector = "dataset"
	}
	leftComp = leftComp.Base().GetSubcomponent(selector)
	rightComp = rightComp.Base().GetSubcomponent(selector)

	leftData, err := leftComp.StructuredData()
	if err != nil {
		return err
	}
	rightData, err := rightComp.StructuredData()
	if err != nil {
		return err
	}

	res.Stat = &deepdiff.Stats{}
	res.A = leftData
	res.B = rightData
	res.Diff, err = deepdiff.Diff(leftData, rightData, deepdiff.OptionSetStats(res.Stat))
	return err
}
