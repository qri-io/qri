package fsi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

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
	// STRemoved is a removed component
	STRemoved = "removed"
)

// StatusItem is a component that has status representation on the filesystem
type StatusItem struct {
	SourceFile string `json:"sourceFile"`
	Component  string `json:"component"`
	Type       string `json:"type"`
	Message    string `json:"message"`
}

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

var componentListOrder = []string{
	"dataset",
	"commit",
	"meta",
	"structure",
	"schema",
	"viz",
	"transform",
	"body",
}

func (si statusItems) Len() int      { return len(si) }
func (si statusItems) Swap(i, j int) { si[i], si[j] = si[j], si[i] }
func (si statusItems) Less(i, j int) bool {
	return componentOrder[si[i].Component] < componentOrder[si[j].Component]
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

	return fsi.StoredStatus(alias)
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
	stored.Commit = nil
	stored.Transform = nil
	stored.Peername = ""

	working, mapping, err := ReadDir(dir)
	if err != nil {
		return nil, err
	}
	working.DropDerivedValues()

	// if err = validate.Dataset(ds); err != nil {
	// 	return nil, fmt.Errorf("dataset is invalid: %s" , err)
	// }

	storedComponents := dsAllComponents(stored)

	for cmpName := range storedComponents {
		// when reporting deletes, ignore "bound" components that must/must-not
		// exist based on external conditions
		if cmpName != componentNameDataset && cmpName != componentNameStructure && cmpName != componentNameCommit && cmpName != componentNameViz {
			cmp := dsComponent(stored, cmpName)
			// If the component was not in the previous version, it can't have been removed.
			if cmp == nil {
				continue
			}
			if _, ok := mapping[cmpName]; !ok {
				change := StatusItem{
					Component: cmpName,
					Type:      STRemoved,
				}
				changes = append(changes, change)
			}
		}
	}

	// Iterate components in a deterministic order, going backwards.
	for i := len(componentListOrder) - 1; i >= 0; i-- {
		path := componentListOrder[i]
		if path == componentNameDataset {
			continue
		}

		localFilepath, ok := mapping[path]
		if !ok {
			continue
		}

		if cmp := dsComponent(stored, path); cmp == nil {
			change := StatusItem{
				SourceFile: localFilepath,
				Component:  path,
				Type:       STAdd,
			}
			changes = append(changes, change)
		} else {

			var storedData []byte
			var workData []byte
			if path == componentNameBody {
				// Getting data for the body works differently.
				if err = stored.OpenBodyFile(fsi.repo.Filesystem()); err != nil {
					return nil, err
				}
				storedBody := stored.BodyFile()
				if storedBody == nil {
					// Handle the case where there's no previous version. Body is "add"ed, do
					// not attempt to read the non-existent stored body.
					change := StatusItem{
						SourceFile: localFilepath,
						Component:  path,
						Type:       STAdd,
					}
					changes = append(changes, change)
					continue
				} else {
					// Read body of previous version.
					defer storedBody.Close()
					storedData, err = ioutil.ReadAll(storedBody)
					if err != nil {
						return nil, err
					}
				}

				workingBody, err := os.Open(filepath.Join(dir, localFilepath))
				if err != nil {
					return nil, err
				}
				defer workingBody.Close()

				workData, err = ioutil.ReadAll(workingBody)
				if err != nil {
					return nil, err
				}
			} else {
				storedData, err = json.Marshal(cmp)
				if err != nil {
					return nil, err
				}

				workData, err = json.Marshal(dsComponent(working, path))
				if err != nil {
					return nil, err
				}
			}

			if !bytes.Equal(storedData, workData) {
				change := StatusItem{
					SourceFile: localFilepath,
					Component:  path,
					Type:       STChange,
				}
				changes = append(changes, change)
			} else {
				change := StatusItem{
					SourceFile: localFilepath,
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

// StoredStatus loads a dataset & presents it in a status-like format
func (fsi *FSI) StoredStatus(refStr string) (changes []StatusItem, err error) {
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

	for cmpName := range dsComponents(stored) {
		si := StatusItem{
			SourceFile: "repo",
			Component:  cmpName,
			Type:       STUnmodified,
		}
		changes = append(changes, si)
	}

	sort.Sort(statusItems(changes))
	return changes, err
}

func dsComponent(ds *dataset.Dataset, cmpName string) interface{} {
	// This switch avoids returning interfaces with nil values and non-nil type tags.
	switch cmpName {
	case componentNameCommit:
		if ds.Commit == nil {
			return nil
		}
		return ds.Commit
	case componentNameDataset:
		return ds
	case componentNameMeta:
		if ds.Meta == nil {
			return nil
		}
		return ds.Meta
	case componentNameSchema:
		if ds.Structure == nil {
			return nil
		}
		return ds.Structure.Schema
	case componentNameBody:
		return ds.BodyPath != ""
	case componentNameStructure:
		if ds.Structure == nil {
			return nil
		}
		return ds.Structure
	case componentNameTransform:
		if ds.Transform == nil {
			return nil
		}
		return ds.Transform
	case componentNameViz:
		if ds.Viz == nil {
			return nil
		}
		return ds.Viz
	default:
		return nil
	}
}

// dsAllComponents returns the components of a dataset as a map of component_name: value
func dsAllComponents(ds *dataset.Dataset) map[string]interface{} {
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
