package base

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
)

var (
	// STUnmodified is "no status"
	STUnmodified = "unmodified"
	// STAdd is an added component
	STAdd = "add"
	// STChange is a modified component
	STChange = "modified"
	// STRemoved is a removed component
	STRemoved = "removed"
	// STParseError is a component that didn't parse
	STParseError = "parse error"
	// STMissing is a component that is missing
	STMissing = "missing"
	// STConflictError is a component with a conflict
	STConflictError = "conflict error"
	// ErrWorkingDirectoryDirty is the error for when the working directory is not clean
	ErrWorkingDirectoryDirty = fmt.Errorf("working directory is dirty")
)

// StatusItem is the status of a component of a dataset, and whether it
// was changed at a specific version in history
type StatusItem struct {
	Component string `json:"component"`
	Type      string `json:"type"`
}

// MarshalJSON marshals a StatusItem
func (si StatusItem) MarshalJSON() ([]byte, error) {
	obj := struct {
		Component string `json:"component"`
		Type      string `json:"type"`
	}{
		Component: si.Component,
		Type:      si.Type,
	}
	return json.Marshal(obj)
}

// ComponentStatus owns functionality to get change status about components
type ComponentStatus struct {
	fs *muxfs.Mux
}

// NewComponentStatus returns a new ComponentStatus
func NewComponentStatus(ctx context.Context, fs *muxfs.Mux) *ComponentStatus {
	return &ComponentStatus{
		fs: fs,
	}
}

// WhatChanged gets changes that happened at a particular version in a dataset's history.
func (cs *ComponentStatus) WhatChanged(ctx context.Context, ref dsref.Ref) (changes []StatusItem, err error) {
	if ref.Path == "" {
		return nil, fmt.Errorf("path is required to determine status at version")
	}

	var next, prev *dataset.Dataset
	if next, err = dsfs.LoadDataset(ctx, cs.fs, ref.Path); err != nil {
		return nil, err
	}

	prevPath := next.PreviousPath
	if prevPath == "" {
		prev = &dataset.Dataset{}
	} else {
		if prev, err = dsfs.LoadDataset(ctx, cs.fs, prevPath); err != nil {
			if strings.Contains(err.Error(), "deadline exceeded") {
				// TODO (b5) - need to handle this situation gracefully, returning an indication
				// that the previous version can't be loaded
				prev = &dataset.Dataset{}
				err = nil
			} else {
				log.Error(err)
				return nil, err
			}
		}
	}

	prevCompCollect := component.ConvertDatasetToComponents(prev, cs.fs)
	prevCompCollect.Base().RemoveSubcomponent("commit")
	prevCompCollect.DropDerivedValues()
	nextCompCollect := component.ConvertDatasetToComponents(next, cs.fs)
	nextCompCollect.Base().RemoveSubcomponent("commit")
	nextCompCollect.DropDerivedValues()

	changes, err = cs.calculateStateTransition(ctx, prevCompCollect, nextCompCollect)
	if err != nil {
		return nil, err
	}
	return changes, nil
}

// calculateStateTransition calculates the differences between two versions of a dataset.
func (cs *ComponentStatus) calculateStateTransition(ctx context.Context, prev, next component.Component) (changes []StatusItem, err error) {

	changes = make([]StatusItem, 0, component.NumberPossibleComponents)

	// See if the dataset itself has a problem.
	dsComp := next.Base().GetSubcomponent("dataset")
	if dsComp != nil && dsComp.Base().ProblemKind != "" {
		changes = append(changes, StatusItem{
			Component: "dataset",
			Type:      dsComp.Base().ProblemKind,
		})
	}

	for _, compName := range component.AllSubcomponentNames() {
		prevComp := prev.Base().GetSubcomponent(compName)
		nextComp := next.Base().GetSubcomponent(compName)

		// Next component might have a problem, such as a parse error, or permission problem.
		if nextComp != nil && nextComp.Base().ProblemKind != "" {
			changes = append(changes, StatusItem{
				Component: compName,
				Type:      nextComp.Base().ProblemKind,
			})
			continue
		}

		if prevComp == nil && nextComp == nil {
			// Didn't exist before, still doesn't - skip this component.
			continue
		} else if prevComp == nil && nextComp != nil {
			// Didn't exist before, does now - component was added.
			changes = append(changes, StatusItem{
				Component: compName,
				Type:      STAdd,
			})
			continue
		} else if prevComp != nil && nextComp == nil {
			// Did exist before, but doesn't now - component was removed.
			changes = append(changes, StatusItem{
				Component: compName,
				Type:      STRemoved,
			})
			continue
		}

		isEqual, err := prevComp.Compare(nextComp)
		if err != nil {
			changes = append(changes, StatusItem{
				Component: compName,
				Type:      STParseError,
			})
			continue
		}

		if isEqual {
			changes = append(changes, StatusItem{
				Component: compName,
				Type:      STUnmodified,
			})
		} else {
			changes = append(changes, StatusItem{
				Component: compName,
				Type:      STChange,
			})
		}
	}

	return changes, nil
}
