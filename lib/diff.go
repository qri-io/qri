package lib

import (
	"context"
	"errors"
	"fmt"

	"github.com/qri-io/dataset/tabular"
	"github.com/qri-io/deepdiff"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	qerr "github.com/qri-io/qri/errors"
)

// DiffMethods encapsulates logic for diffing Datasets on Qri
type DiffMethods struct {
	d dispatcher
}

// Name returns the name of this method group
func (m DiffMethods) Name() string {
	return "diff"
}

// Attributes defines attributes for each method
func (m DiffMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"changes": {Endpoint: AEChanges, HTTPVerb: "POST"},
		"diff":    {Endpoint: AEDiff, HTTPVerb: "POST"},
	}
}

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
	LeftSide  string `schema:"leftPath" json:"leftPath" qri:"dsrefOrFspath"`
	RightSide string `schema:"rightPath" json:"rightPath" qri:"dsrefOrFspath"`
	// If not null, the working directory that the diff is using
	WorkingDir string `qri:"fspath"`
	// Whether to get the previous version of the left parameter
	UseLeftPrevVersion bool

	// Which component or part of a dataset to compare
	Selector string
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
func (m DiffMethods) Diff(ctx context.Context, p *DiffParams) (*DiffResponse, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "diff"), p)
	if res, ok := got.(*DiffResponse); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

func schemaDiff(ctx context.Context, left, right *component.BodyComponent) ([]*Delta, *DiffStat, error) {
	dd := deepdiff.New()
	if left.Format == ".csv" && right.Format == ".csv" {
		left, _, err := tabular.ColumnsFromJSONSchema(left.InferredSchema)
		if err != nil {
			return nil, nil, err
		}

		right, _, err := tabular.ColumnsFromJSONSchema(right.InferredSchema)
		if err != nil {
			return nil, nil, err
		}

		return dd.StatDiff(ctx, left.Titles(), right.Titles())
	}
	return dd.StatDiff(ctx, left.InferredSchema, right.InferredSchema)
}

// assume a non-empty string, which isn't a dataset reference, is a file
func isFilePath(text string) bool {
	if text == "" {
		return false
	}
	return !dsref.IsRefString(text)
}

// diffImpl holds the method implementations for DiffMethods
type diffImpl struct{}

// Diff computes the diff of two source
func (diffImpl) Diff(scope scope, p *DiffParams) (*DiffResponse, error) {
	res := &DiffResponse{}

	diffMode, err := p.diffMode()
	if err != nil {
		return nil, err
	}

	if diffMode == FilepathDiffMode {
		// Compare body files.
		leftComp := component.NewBodyComponent(p.LeftSide)
		leftData, err := leftComp.StructuredData()
		if err != nil {
			return nil, err
		}

		rightComp := component.NewBodyComponent(p.RightSide)
		rightData, err := rightComp.StructuredData()
		if err != nil {
			return nil, err
		}

		res.Schema, res.SchemaStat, err = schemaDiff(scope.Context(), leftComp, rightComp)
		if err != nil {
			return nil, err
		}

		dd := deepdiff.New()
		res.Diff, res.Stat, err = dd.StatDiff(scope.Context(), leftData, rightData)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	// Left side of diff loaded into a component
	ds, err := scope.Loader().LoadDataset(scope.Context(), p.LeftSide)
	if err != nil {
		if errors.Is(err, dsref.ErrNoHistory) {
			return nil, qerr.New(err, fmt.Sprintf("dataset %s has no versions, nothing to diff against", p.LeftSide))
		}
		return nil, err
	}
	// TODO (b5) - setting name & peername to zero values makes tests pass, but
	// calling ds.DropDerivedValues is overzealous. investigate the right solution
	ds.Name = ""
	ds.Peername = ""
	leftComp := component.ConvertDatasetToComponents(ds, scope.Filesystem())

	// Right side of diff laoded into a component
	var rightComp component.Component

	switch diffMode {
	case WorkingDirectoryDiffMode:
		// Working directory, read dataset from the current files.
		rightComp, err = component.ListDirectoryComponents(p.WorkingDir)
		if err != nil {
			return nil, err
		}
		err = component.ExpandListedComponents(rightComp, scope.Filesystem())
		if err != nil {
			return nil, err
		}
		// TODO(dlong): Hack! This is what fills the value. StucturedData assumes this has been
		// called. Should cleanup component's API so that this isn't necessary.
		_, err = component.ToDataset(rightComp)
		if err != nil {
			return nil, err
		}
	case PrevVersionDiffMode:
		// The head version was already loaded, use that for the right side of the diff
		rightComp = leftComp
		// Load previous dataset version for the new left side
		if ds.PreviousPath == "" {
			return nil, fmt.Errorf("dataset has only one version, nothing to diff against")
		}
		ds, err = dsfs.LoadDataset(scope.Context(), scope.Filesystem(), ds.PreviousPath)
		if err != nil {
			return nil, err
		}
		leftComp = component.ConvertDatasetToComponents(ds, scope.Filesystem())
	case DatasetRefDiffMode:
		ds, err = scope.Loader().LoadDataset(scope.Context(), p.RightSide)
		if err != nil {
			return nil, err
		}
		// TODO (b5) - setting name & peername to zero values makes tests pass, but
		// calling ds.DropDerivedValues is overzealous. investigate the right solution
		ds.Name = ""
		ds.Peername = ""
		rightComp = component.ConvertDatasetToComponents(ds, scope.Filesystem())
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
						return nil, err
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
						return nil, err
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
		return nil, fmt.Errorf("component %q not found", selector)
	}

	leftData, err := leftComp.StructuredData()
	if err != nil {
		return nil, err
	}
	rightData, err := rightComp.StructuredData()
	if err != nil {
		return nil, err
	}

	dd := deepdiff.New()
	res.Diff, res.Stat, err = dd.StatDiff(scope.Context(), leftData, rightData)
	if err != nil {
		return nil, err
	}
	return res, nil
}
