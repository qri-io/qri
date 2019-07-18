package fsi

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	// "github.com/qri-io/dataset/validate"
)

var (
	// STUnmodified is "no status"
	STUnmodified = "unmodified"
	// STAdd is an added component
	STAdd = "add"
	// STChange is a modified component
	STChange = "modified"
	// STRemove is a removed component, currently not really supported?
	STRemove = "remove"
)

// StatusItem is a component that has status representation on the filesystem
type StatusItem struct {
	SourceFile string `json:"sourceFile"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	Message    string `json:"message"`
}

// AliasStatus returns the status for a given dataset alias
func (fsi *FSI) AliasStatus(alias string) (changes []StatusItem, err error) {
	links, err := fsi.load()
	if err != nil {
		return nil, err
	}

	for _, l := range links {
		if l.Alias == alias {
			return fsi.Status(l.Path)
		}
	}

	return nil, fmt.Errorf("alias not found: %s", alias)
}

// Status reads the diff status from the current working directory
func (fsi *FSI) Status(dir string) (changes []StatusItem, err error) {
	refStr, ok := GetLinkedFilesysRef(dir)
	if !ok {
		err = fmt.Errorf("not a linked directory")
		return nil, err
	}

	ref, err := repo.ParseDatasetRef(refStr)
	if err != nil {
		return nil, err
	}

	var stored *dataset.Dataset
	if err := repo.CanonicalizeDatasetRef(fsi.repo, &ref); err != nil {
		if err == repo.ErrNotFound {
			// no dataset, compare to an empty ds
			stored = &dataset.Dataset{}
		} else {
			return nil, err
		}
	} else {
		if stored, err = dsfs.LoadDataset(fsi.repo.Store(), ref.Path); err != nil {
			return nil, err
		}
	}

	// stored.DropDerivedValues()

	ds, mapping, err := ReadDir(dir)
	if err != nil {
		return nil, err
	}
	// ds.DropDerivedValues()

	// if err = validate.Dataset(ds); err != nil {
	// 	return nil, fmt.Errorf("dataset is invalid: %s" , err)
	// }

	for path, sourceFilepath := range mapping {
		if cmp := getComponent(stored, path); cmp == nil {
			change := StatusItem{
				SourceFile: sourceFilepath,
				Path:       path,
				Type:       STAdd,
			}
			changes = append(changes, change)
		} else {
			srcData, err := json.Marshal(cmp)
			if err != nil {
				return nil, err
			}
			wdData, err := json.Marshal(getComponent(ds, path))
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(srcData, wdData) {
				change := StatusItem{
					SourceFile: sourceFilepath,
					Path:       path,
					Type:       STChange,
				}
				changes = append(changes, change)
			} else {
				change := StatusItem{
					SourceFile: sourceFilepath,
					Path:       path,
					Type:       STUnmodified,
				}
				changes = append(changes, change)
			}
		}
	}

	return changes, nil
}

func getComponent(ds *dataset.Dataset, cmpName string) interface{} {
	switch cmpName {
	case componentNameCommit:
		return ds.Commit
	case componentNameDataset:
		return ds
	case componentNameMeta:
		return ds.Meta
	case componentNameSchema:
		if ds.Structure == nil {
			return nil
		}
		return ds.Structure.Schema
	case componentNameBody:
		// TODO (b5) - this isn't going to work properly
		return ds.Body
	case componentNameStructure:
		return ds.Structure
	case componentNameTransform:
		return ds.Transform
	case componentNameViz:
		return ds.Viz
	default:
		return nil
	}
}
