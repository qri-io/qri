package fsi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	// "github.com/qri-io/dataset/validate"
)

// statusItems is a slice of component Status, used for sorting
type statusItems []StatusItem

// componentOrder is the canonical order for components. values are negative
// such that they will report less than a nonexistent key, which will return 0
// and report after specified keys
var componentOrder = map[string]int{
	"dataset":   -8,
	"commit":    -7,
	"meta":      -6,
	"structure": -5,
	"schema":    -4,
	"viz":       -3,
	"transform": -2,
	"body":      -1,
}

func (si statusItems) Len() int      { return len(si) }
func (si statusItems) Swap(i, j int) { si[i], si[j] = si[j], si[i] }
func (si statusItems) Less(i, j int) bool {
	return componentOrder[si[i].Component] < componentOrder[si[j].Component]
}

var (
	// STUnmodified is "no status"
	STUnmodified = "unmodified"
	// STAdd is an added component
	STAdd = "add"
	// STChange is a modified component
	STChange = "modified"
	// STRemoved is a removed component, currently not really supported?
	STRemoved = "removed"
)

// StatusItem is a component that has status representation on the filesystem
type StatusItem struct {
	SourceFile string `json:"sourceFile"`
	Component  string `json:"component"`
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

	stored.DropDerivedValues()

	ds, mapping, err := ReadDir(dir)
	if err != nil {
		return nil, err
	}
	ds.DropDerivedValues()

	// if err = validate.Dataset(ds); err != nil {
	// 	return nil, fmt.Errorf("dataset is invalid: %s" , err)
	// }

	storedComponents := dsComponents(stored)

	for cmpName := range storedComponents {
		// when reporting deletes, ignore "bound" components that must/must-not
		// exist based on external conditions
		if cmpName != componentNameDataset && cmpName != componentNameStructure && cmpName != componentNameCommit {
			if _, ok := mapping[cmpName]; !ok {
				change := StatusItem{
					Component: cmpName,
					Type:      STRemoved,
				}
				changes = append(changes, change)
			}
		}
	}

	for path, sourceFilepath := range mapping {
		if cmp := dsComponent(stored, path); cmp == nil {
			change := StatusItem{
				SourceFile: sourceFilepath,
				Component:  path,
				Type:       STAdd,
			}
			changes = append(changes, change)
		} else {
			srcData, err := json.Marshal(cmp)
			if err != nil {
				return nil, err
			}
			wdData, err := json.Marshal(dsComponent(ds, path))
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(srcData, wdData) {
				change := StatusItem{
					SourceFile: sourceFilepath,
					Component:  path,
					Type:       STChange,
				}
				changes = append(changes, change)
			} else {
				change := StatusItem{
					SourceFile: sourceFilepath,
					Component:  path,
					Type:       STUnmodified,
				}
				changes = append(changes, change)
			}
		}
	}

	sort.Sort(statusItems(changes))
	return changes, nil
}

func dsComponent(ds *dataset.Dataset, cmpName string) interface{} {
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

// dsComponents returns the components of a dataset as a map of component_name: value
func dsComponents(ds *dataset.Dataset) map[string]interface{} {
	components := map[string]interface{}{}
	cmpNames := []string{
		componentNameCommit,
		componentNameDataset,
		componentNameMeta,
		componentNameSchema,
		componentNameBody,
		componentNameStructure,
		componentNameTransform,
		componentNameViz,
	}

	for _, cmpName := range cmpNames {
		if cmp := dsComponent(ds, cmpName); cmp != nil {
			components[cmpName] = cmp
		}
	}

	return components
}
