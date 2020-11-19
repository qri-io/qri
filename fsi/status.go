package fsi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
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
	// STConflictError is a component with a conflict
	STConflictError = "conflict error"
	// ErrWorkingDirectoryDirty is the error for when the working directory is not clean
	ErrWorkingDirectoryDirty = fmt.Errorf("working directory is dirty")
)

// StatusItem is a component that has status representation on the filesystem
type StatusItem struct {
	SourceFile string    `json:"sourceFile"`
	Component  string    `json:"component"`
	Type       string    `json:"type"`
	Message    string    `json:"message"`
	Mtime      time.Time `json:"mtime"`
}

// MarshalJSON marshals a StatusItem, handling mtime specially
func (si StatusItem) MarshalJSON() ([]byte, error) {
	obj := struct {
		SourceFile string `json:"sourceFile"`
		Component  string `json:"component"`
		Type       string `json:"type"`
		Message    string `json:"message"`
		Mtime      string `json:"mtime,omitempty"`
	}{
		SourceFile: si.SourceFile,
		Component:  si.Component,
		Type:       si.Type,
		Message:    si.Message,
		Mtime:      si.Mtime.Format(time.RFC3339),
	}
	return json.Marshal(obj)
}

// Status compares status of the current working directory against the dataset's last version
func (fsi *FSI) Status(ctx context.Context, dir string) (changes []StatusItem, err error) {
	fs := fsi.repo.Filesystem()
	ref, ok := GetLinkedFilesysRef(dir)
	if !ok {
		err = fmt.Errorf("not a linked directory")
		return nil, err
	}

	var stored *dataset.Dataset
	vi, err := repo.GetVersionInfoShim(fsi.repo, ref)
	if err != nil {
		return nil, err
	}
	if vi.Path == "" {
		// no dataset, compare to an empty ds
		stored = &dataset.Dataset{}
	} else {
		if stored, err = dsfs.LoadDataset(ctx, fs, vi.Path); err != nil {
			return nil, err
		}
	}

	stored.DropDerivedValues()
	stored.Commit = nil
	stored.Peername = ""

	working, err := component.ListDirectoryComponents(dir)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	err = component.ExpandListedComponents(working, fs)
	if err != nil {
		return nil, err
	}

	// TODO: If in the future we cache mtimes and previous status, we can more lazily read only
	// some components.

	prevComps := component.ConvertDatasetToComponents(stored, fs)
	nextComps := working
	return fsi.CalculateStateTransition(ctx, prevComps, nextComps)
}

// CalculateStateTransition calculates the differences between two versions of a dataset.
func (fsi *FSI) CalculateStateTransition(ctx context.Context, prev, next component.Component) (changes []StatusItem, err error) {

	changes = make([]StatusItem, 0, component.NumberPossibleComponents)

	// See if the dataset itself has a problem.
	dsComp := next.Base().GetSubcomponent("dataset")
	if dsComp != nil && dsComp.Base().ProblemKind != "" {
		changes = append(changes, StatusItem{
			SourceFile: dsComp.Base().SourceFile,
			Component:  "dataset",
			Type:       dsComp.Base().ProblemKind,
			Mtime:      dsComp.Base().ModTime,
		})
	}

	for _, compName := range component.AllSubcomponentNames() {
		prevComp := prev.Base().GetSubcomponent(compName)
		nextComp := next.Base().GetSubcomponent(compName)

		// Next component might have a problem, such as a parse error, or permission problem.
		if nextComp != nil && nextComp.Base().ProblemKind != "" {
			changes = append(changes, StatusItem{
				SourceFile: nextComp.Base().SourceFile,
				Component:  compName,
				Type:       nextComp.Base().ProblemKind,
				Mtime:      nextComp.Base().ModTime,
			})
			continue
		}

		if prevComp == nil && nextComp == nil {
			// Didn't exist before, still doesn't - skip this component.
			continue
		} else if prevComp == nil && nextComp != nil {
			// Didn't exist before, does now - component was added.
			changes = append(changes, StatusItem{
				SourceFile: nextComp.Base().SourceFile,
				Component:  compName,
				Type:       STAdd,
				Mtime:      nextComp.Base().ModTime,
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
				SourceFile: nextComp.Base().SourceFile,
				Component:  compName,
				Type:       STParseError,
				Mtime:      nextComp.Base().ModTime,
			})
			continue
		}

		if isEqual {
			changes = append(changes, StatusItem{
				SourceFile: nextComp.Base().SourceFile,
				Component:  compName,
				Type:       STUnmodified,
				Mtime:      nextComp.Base().ModTime,
			})
		} else {
			changes = append(changes, StatusItem{
				SourceFile: nextComp.Base().SourceFile,
				Component:  compName,
				Type:       STChange,
				Mtime:      nextComp.Base().ModTime,
			})
		}
	}

	return changes, nil
}

// StatusAtVersion gets changes that happened at a particular version in a dataset's history.
func (fsi *FSI) StatusAtVersion(ctx context.Context, ref dsref.Ref) (changes []StatusItem, err error) {
	fs := fsi.repo.Filesystem()
	if ref.Path == "" {
		return nil, fmt.Errorf("path is required to determine status at version")
	}

	var next, prev *dataset.Dataset
	if next, err = dsfs.LoadDataset(ctx, fs, ref.Path); err != nil {
		return nil, err
	}

	prevPath := next.PreviousPath
	if prevPath == "" {
		prev = &dataset.Dataset{}
	} else {
		if prev, err = dsfs.LoadDataset(ctx, fs, prevPath); err != nil {
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

	prevCompCollect := component.ConvertDatasetToComponents(prev, fs)
	prevCompCollect.Base().RemoveSubcomponent("commit")
	prevCompCollect.DropDerivedValues()
	nextCompCollect := component.ConvertDatasetToComponents(next, fs)
	nextCompCollect.Base().RemoveSubcomponent("commit")
	nextCompCollect.DropDerivedValues()

	changes, err = fsi.CalculateStateTransition(ctx, prevCompCollect, nextCompCollect)
	if err != nil {
		return nil, err
	}
	for i, ch := range changes {
		comp := ch.Component
		if comp == "meta" || comp == "body" || comp == "structure" {
			changes[i].SourceFile = comp
		}
	}
	return changes, nil
}

// IsWorkingDirectoryClean returns nil if the directory is clean, or ErrWorkingDirectoryDirty if
// it is dirty
func (fsi *FSI) IsWorkingDirectoryClean(ctx context.Context, dir string) error {
	changes, err := fsi.Status(ctx, dir)
	if err != nil {
		return err
	}
	for _, ch := range changes {
		if ch.Type != STUnmodified {
			return ErrWorkingDirectoryDirty
		}
	}
	return nil

}
