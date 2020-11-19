package lib

import (
	"context"
	"errors"
	"fmt"

	"github.com/qri-io/deepdiff"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	qerr "github.com/qri-io/qri/errors"
)

// Delta is an alias for deepdiff.Delta, abstracting the deepdiff implementation
// away from packages that depend on lib
type Delta = deepdiff.Delta

// DiffStat is an alias for deepdiff.Stat, abstracting the deepdiff implementation
// away from packages that depend on lib
type DiffStat = deepdiff.Stats

// DiffParams defines parameters for diffing two sources. There are three valid ways to use these
// parameters: 1) both LeftSide and RightSide set, 2) only LeftSide set with a WorkingDir, 3) only
// LeftSide set with the UseLeftPrevVersion flag.
type DiffParams struct {
	// File paths or reference to datasets
	LeftSide, RightSide string
	// If not null, the working directory that the diff is using
	WorkingDir string
	// Whether to get the previous version of the left parameter
	UseLeftPrevVersion bool

	// Which component or part of a dataset to compare
	Selector string

	Remote string
}

// diffMode determinse
func (p *DiffParams) diffMode() (DiffMode, error) {
	// Check parameters to make sure they fit one of the three cases that diff allows.
	if p.LeftSide == "" && p.RightSide == "" {
		return InvalidDiffMode, fmt.Errorf("nothing to diff")
	} else if p.LeftSide != "" && p.RightSide != "" {
		// Have two string parameters to compare. Should either both be references, or neither
		// be references.
		diffMode := InvalidDiffMode
		if dsref.IsRefString(p.LeftSide) && dsref.IsRefString(p.RightSide) {
			diffMode = DatasetRefDiffMode
		} else if isFilePath(p.LeftSide) && isFilePath(p.RightSide) {
			diffMode = FilepathDiffMode
		} else {
			return InvalidDiffMode, fmt.Errorf("cannot compare a file to dataset, must compare similar things")
		}
		// Neither of the flags should be set.
		if p.WorkingDir != "" {
			return diffMode, fmt.Errorf("cannot use working directory when comparing two sources")
		}
		if p.UseLeftPrevVersion {
			return diffMode, fmt.Errorf("cannot use previous version when comparing two sources")
		}
		return diffMode, nil
	} else if dsref.IsRefString(p.LeftSide) && p.WorkingDir != "" {
		// Comparing the contents of a working directory to the dataset it represents
		// TODO(dustmop): Should verify that the working directory *matches* the dataset
		if p.UseLeftPrevVersion {
			return InvalidDiffMode, fmt.Errorf("cannot use both previous version and working directory")
		}
		return WorkingDirectoryDiffMode, nil
	} else if dsref.IsRefString(p.LeftSide) && p.UseLeftPrevVersion {
		// Comparing a dataset to its previous version
		if p.WorkingDir != "" {
			return InvalidDiffMode, fmt.Errorf("cannot use both previous version and working directory")
		}
		return PrevVersionDiffMode, nil
	}
	return InvalidDiffMode, fmt.Errorf("invalid parameters to diff")
}

// DiffResponse is the result of a call to diff
type DiffResponse struct {
	Stat       *DiffStat `json:"stat,omitempty"`
	SchemaStat *DiffStat `json:"schemaStat,omitempty"`
	Schema     []*Delta  `json:"schema,omitempty"`
	Diff       []*Delta  `json:"diff,omitempty"`
}

// DiffMode is one of the methods that diff can perform
type DiffMode int

const (
	// InvalidDiffMode is the default diff mode
	InvalidDiffMode DiffMode = iota
	// DatasetRefDiffMode will diff two dataset references
	DatasetRefDiffMode
	// FilepathDiffMode will diff two files
	FilepathDiffMode
	// WorkingDirectoryDiffMode will diff a working directory against its dataset head
	WorkingDirectoryDiffMode
	// PrevVersionDiffMode will diff a dataset head against its previous version
	PrevVersionDiffMode
)

// Diff computes the diff of two sources
func (m *DatasetMethods) Diff(p *DiffParams, res *DiffResponse) (err error) {
	// absolutize any local paths before a possible trip over RPC to another local process
	if !dsref.IsRefString(p.LeftSide) {
		if err = qfs.AbsPath(&p.LeftSide); err != nil {
			return err
		}
	}
	if !dsref.IsRefString(p.RightSide) {
		if err = qfs.AbsPath(&p.RightSide); err != nil {
			return err
		}
	}

	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.Diff", p, res))
	}
	ctx := context.TODO()

	diffMode, err := p.diffMode()
	if err != nil {
		return err
	}

	if diffMode == FilepathDiffMode {
		// Compare body files.
		leftComp := component.NewBodyComponent(p.LeftSide)
		leftData, err := leftComp.StructuredData()
		if err != nil {
			return err
		}

		rightComp := component.NewBodyComponent(p.RightSide)
		rightData, err := rightComp.StructuredData()
		if err != nil {
			return err
		}

		res.Schema, res.SchemaStat, err = schemaDiff(ctx, leftComp, rightComp)
		if err != nil {
			return err
		}

		dd := deepdiff.New()
		res.Diff, res.Stat, err = dd.StatDiff(ctx, leftData, rightData)
		return err
	}

	// Left side of diff loaded into a component
	parseResolveLoad, err := m.inst.NewParseResolveLoadFunc(p.Remote)
	if err != nil {
		return err
	}
	ds, err := parseResolveLoad(ctx, p.LeftSide)
	if err != nil {
		if errors.Is(err, dsref.ErrNoHistory) {
			return qerr.New(err, fmt.Sprintf("dataset %s has no versions, nothing to diff against", p.LeftSide))
		}
		return err
	}
	// TODO (b5) - setting name & peername to zero values makes tests pass, but
	// calling ds.DropDerivedValues is overzealous. investigate the right solution
	ds.Name = ""
	ds.Peername = ""
	leftComp := component.ConvertDatasetToComponents(ds, m.inst.repo.Filesystem())

	// Right side of diff laoded into a component
	var rightComp component.Component

	switch diffMode {
	case WorkingDirectoryDiffMode:
		// Working directory, read dataset from the current files.
		rightComp, err = component.ListDirectoryComponents(p.WorkingDir)
		if err != nil {
			return err
		}
		err = component.ExpandListedComponents(rightComp, m.inst.repo.Filesystem())
		if err != nil {
			return err
		}
		// TODO(dlong): Hack! This is what fills the value. StucturedData assumes this has been
		// called. Should cleanup component's API so that this isn't necessary.
		_, err = component.ToDataset(rightComp)
		if err != nil {
			return err
		}
	case PrevVersionDiffMode:
		// The head version was already loaded, use that for the right side of the diff
		rightComp = leftComp
		// Load previous dataset version for the new left side
		if ds.PreviousPath == "" {
			return fmt.Errorf("dataset has only one version, nothing to diff against")
		}
		ds, err = dsfs.LoadDataset(ctx, m.inst.repo.Filesystem(), ds.PreviousPath)
		if err != nil {
			return err
		}
		leftComp = component.ConvertDatasetToComponents(ds, m.inst.repo.Filesystem())
	case DatasetRefDiffMode:
		ds, err = parseResolveLoad(ctx, p.RightSide)
		if err != nil {
			return err
		}
		// TODO (b5) - setting name & peername to zero values makes tests pass, but
		// calling ds.DropDerivedValues is overzealous. investigate the right solution
		ds.Name = ""
		ds.Peername = ""
		rightComp = component.ConvertDatasetToComponents(ds, m.inst.repo.Filesystem())
	}

	// If in an FSI linked working directory, drop derived values, since the user is not
	// expected to have those transient values on their checked out files.
	if diffMode == WorkingDirectoryDiffMode {
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
	if leftComp == nil || rightComp == nil {
		return fmt.Errorf("component %q not found", selector)
	}

	leftData, err := leftComp.StructuredData()
	if err != nil {
		return err
	}
	rightData, err := rightComp.StructuredData()
	if err != nil {
		return err
	}

	dd := deepdiff.New()
	res.Diff, res.Stat, err = dd.StatDiff(ctx, leftData, rightData)
	return err
}

func schemaDiff(ctx context.Context, left, right *component.BodyComponent) ([]*Delta, *DiffStat, error) {
	dd := deepdiff.New()
	if left.Format == ".csv" && right.Format == ".csv" {
		left, err := terribleHackToGetHeaderRow(left.InferredSchema)
		if err != nil {
			return nil, nil, err
		}

		right, err := terribleHackToGetHeaderRow(right.InferredSchema)
		if err != nil {
			return nil, nil, err
		}

		return dd.StatDiff(ctx, left, right)
	}
	return dd.StatDiff(ctx, left.InferredSchema, right.InferredSchema)
}

// TODO (b5) - this is terrible. We need better logic error handling for
// jsonschemas describing CSV data. We're relying too heavily on the schema
// being well-formed
func terribleHackToGetHeaderRow(sch map[string]interface{}) ([]string, error) {
	if itemObj, ok := sch["items"].(map[string]interface{}); ok {
		if itemArr, ok := itemObj["items"].([]interface{}); ok {
			titles := make([]string, len(itemArr))
			for i, f := range itemArr {
				if field, ok := f.(map[string]interface{}); ok {
					if title, ok := field["title"].(string); ok {
						titles[i] = title
					}
				}
			}
			return titles, nil
		}
	}
	log.Debug("that terrible hack to detect header row & types just failed")
	return nil, fmt.Errorf("nope")
}

// assume a non-empty string, which isn't a dataset reference, is a file
func isFilePath(text string) bool {
	if text == "" {
		return false
	}
	return !dsref.IsRefString(text)
}
