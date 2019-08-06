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
	"github.com/qri-io/qfs"
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
	// STParseError is a component that didn't parse
	STParseError = "parse error"
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
	ref, err := fsi.getRepoRef(alias)
	if err != nil {
		return nil, err
	}

	if ref.FSIPath != "" {
		return fsi.Status(ref.FSIPath)
	}

	return fsi.StoredStatus(ref.String())
}

// Status reads the diff status from the current working directory
func (fsi *FSI) Status(dir string) (changes []StatusItem, err error) {
	refStr, ok := GetLinkedFilesysRef(dir)
	if !ok {
		err = fmt.Errorf("not a linked directory")
		return nil, err
	}

	var stored *dataset.Dataset
	ref, err := fsi.getRepoRef(refStr)
	if ref.Path == "" {
		// no dataset, compare to an empty ds
		stored = &dataset.Dataset{}
	} else {
		if stored, err = dsfs.LoadDataset(fsi.repo.Store(), ref.Path); err != nil {
			return nil, err
		}
	}

	stored.DropDerivedValues()
	stored.Commit = nil
	stored.Transform = nil
	stored.Peername = ""

	working, fileMap, problems, err := ReadDir(dir)
	if err != nil {
		return nil, err
	}
	working.DropDerivedValues()

	// Set body file from local filesystem.
	if bodyFilename, ok := fileMap[componentNameBody]; ok {
		bf, err := os.Open(filepath.Join(dir, bodyFilename))
		if err != nil {
			return nil, err
		}
		working.SetBodyFile(qfs.NewMemfileReader(bodyFilename, bf))
	}

	return fsi.CalculateStateTransition(stored, working, fileMap, problems)
}

// CalculateStateTransition calculates the differences between two versions of a dataset.
func (fsi *FSI) CalculateStateTransition(prev, next *dataset.Dataset, fileMap, problems map[string]string) (changes []StatusItem, err error) {
	// if err = validate.Dataset(ds); err != nil {
	// 	return nil, fmt.Errorf("dataset is invalid: %s" , err)
	// }

	// Problems is nil unless some components have errors
	if problems != nil {
		for i := 0; i < len(componentListOrder); i++ {
			cmpName := componentListOrder[i]
			if cmpFilename, ok := problems[cmpName]; ok {
				change := StatusItem{
					SourceFile: cmpFilename,
					Component:  cmpName,
					Type:       STParseError,
				}
				changes = append(changes, change)
			}
		}
	}

	prevComponents := dsAllComponents(prev)

	for cmpName := range prevComponents {
		// when reporting deletes, ignore "bound" components that must/must-not
		// exist based on external conditions
		if cmpName != componentNameDataset && cmpName != componentNameStructure && cmpName != componentNameCommit && cmpName != componentNameViz {

			// Skip adding `removed` messages if we already added `problem` for this component.
			if problems != nil {
				if _, ok := problems[cmpName]; ok {
					continue
				}
			}

			cmp := dsComponent(prev, cmpName)
			// If the component was not in the previous version, it can't have been removed.
			if cmp == nil {
				continue
			}
			if _, ok := fileMap[cmpName]; !ok {
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

		localFilepath, ok := fileMap[path]
		if !ok {
			continue
		}

		// Special case: if the schema had a parse error, skip checking the structure. Otherwise,
		// the structure will be shown as "modified".
		// TODO(dlong): Handle the case where structure actually does have changes but schema
		// has a parse error.
		if problems != nil {
			if path == componentNameStructure {
				if _, ok := problems[componentNameSchema]; ok {
					continue
				}
			}
		}

		if cmp := dsComponent(prev, path); cmp == nil {
			change := StatusItem{
				SourceFile: localFilepath,
				Component:  path,
				Type:       STAdd,
			}
			changes = append(changes, change)
		} else {

			var prevData []byte
			var nextData []byte
			if path == componentNameBody {
				// Getting data for the body works differently.
				if err = prev.OpenBodyFile(fsi.repo.Filesystem()); err != nil {
					return nil, err
				}
				prevBody := prev.BodyFile()
				if prevBody == nil {
					// Handle the case where there's no previous version. Body is "add"ed, do
					// not attempt to read the non-existent body.
					change := StatusItem{
						SourceFile: localFilepath,
						Component:  path,
						Type:       STAdd,
					}
					changes = append(changes, change)
					continue
				} else {
					// Read body of previous version.
					defer prevBody.Close()
					prevData, err = ioutil.ReadAll(prevBody)
					if err != nil {
						return nil, err
					}
				}

				// Getting data for the body works differently.
				if err = next.OpenBodyFile(fsi.repo.Filesystem()); err != nil {
					return nil, err
				}
				nextBody := next.BodyFile()
				// TODO(dlong): Handle case where neither version has a body / body is removed.
				if nextBody != nil {
					// Read body of next version.
					defer nextBody.Close()
					nextData, err = ioutil.ReadAll(nextBody)
					if err != nil {
						return nil, err
					}
				}
			} else {
				prevData, err = json.Marshal(cmp)
				if err != nil {
					return nil, err
				}

				nextData, err = json.Marshal(dsComponent(next, path))
				if err != nil {
					return nil, err
				}
			}

			if !bytes.Equal(prevData, nextData) {
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

	var next, prev *dataset.Dataset
	if err := repo.CanonicalizeDatasetRef(fsi.repo, &ref); err != nil {
		return nil, err
	}
	if next, err = dsfs.LoadDataset(fsi.repo.Store(), ref.Path); err != nil {
		return nil, err
	}

	prevPath := next.PreviousPath
	if prevPath == "" {
		prev = &dataset.Dataset{}
	} else {
		if prev, err = dsfs.LoadDataset(fsi.repo.Store(), prevPath); err != nil {
			return nil, err
		}
	}

	fileMap := make(map[string]string)
	if next.Meta != nil {
		fileMap["meta"] = "meta"
	}
	if next.BodyPath != "" || next.BodyFile() != nil {
		fileMap["body"] = "body"
	}
	if next.Structure != nil && next.Structure.Schema != nil {
		fileMap["schema"] = "schema"
	}
	return fsi.CalculateStateTransition(prev, next, fileMap, nil)
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
