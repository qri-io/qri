package fsi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qfs"
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

// AliasToLinkedDir converts the given dataset alias to the FSI path it is linked to.
func (fsi *FSI) AliasToLinkedDir(alias string) (string, error) {
	ref, err := fsi.getRepoRef(alias)
	if err != nil && err != repo.ErrNoHistory {
		return "", err
	}
	if ref.FSIPath == "" {
		return "", fmt.Errorf("StatusForAlias may only be used with linked datasets")
	}
	return ref.FSIPath, nil
}

// Status compares status of the current working directory against the dataset's last version
func (fsi *FSI) Status(ctx context.Context, dir string) (changes []StatusItem, err error) {
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
		if stored, err = dsfs.LoadDataset(ctx, fsi.repo.Store(), ref.Path); err != nil {
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
	if bodyStat, ok := fileMap[componentNameBody]; ok {
		bf, err := os.Open(bodyStat.Path)
		if err != nil {
			return nil, err
		}
		working.SetBodyFile(qfs.NewMemfileReader(bodyStat.Path, bf))
	}

	return fsi.CalculateStateTransition(ctx, stored, working, fileMap, problems)
}

// CalculateStateTransition calculates the differences between two versions of a dataset.
func (fsi *FSI) CalculateStateTransition(ctx context.Context, prev, next *dataset.Dataset, fileMap, problems map[string]FileStat) (changes []StatusItem, err error) {
	// if err = validate.Dataset(ds); err != nil {
	// 	return nil, fmt.Errorf("dataset is invalid: %s" , err)
	// }

	// Problems is nil unless some components have errors
	if problems != nil {
		for i := 0; i < len(componentListOrder); i++ {
			cmpName := componentListOrder[i]
			if cmpFile, ok := problems[cmpName]; ok {
				change := StatusItem{
					SourceFile: cmpFile.Path,
					Component:  cmpName,
					Type:       STParseError,
					Mtime:      cmpFile.Mtime,
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

		localFile, ok := fileMap[path]
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
				SourceFile: localFile.Path,
				Component:  path,
				Type:       STAdd,
				Mtime:      localFile.Mtime,
			}
			changes = append(changes, change)
		} else {

			var prevData []byte
			var nextData []byte
			if path == componentNameBody {
				// Getting data for the body works differently.
				if err = prev.OpenBodyFile(ctx, fsi.repo.Filesystem()); err != nil {
					return nil, err
				}
				prevBody := prev.BodyFile()
				if prevBody == nil {
					// Handle the case where there's no previous version. Body is "add"ed, do
					// not attempt to read the non-existent body.
					change := StatusItem{
						SourceFile: localFile.Path,
						Component:  path,
						Type:       STAdd,
						Mtime:      localFile.Mtime,
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
				if err = next.OpenBodyFile(ctx, fsi.repo.Filesystem()); err != nil {
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
					SourceFile: localFile.Path,
					Component:  path,
					Type:       STChange,
					Mtime:      localFile.Mtime,
				}
				changes = append(changes, change)
			} else {
				change := StatusItem{
					SourceFile: localFile.Path,
					Component:  path,
					Type:       STUnmodified,
					Mtime:      localFile.Mtime,
				}
				changes = append(changes, change)
			}
		}
	}

	sort.Sort(statusItems(changes))
	return changes, nil
}

// StatusAtVersion gets changes that happened at a particular version in a dataset's history.
func (fsi *FSI) StatusAtVersion(ctx context.Context, refStr string) (changes []StatusItem, err error) {
	ref, err := repo.ParseDatasetRef(refStr)
	if err != nil {
		return nil, err
	}

	var next, prev *dataset.Dataset
	if err := repo.CanonicalizeDatasetRef(fsi.repo, &ref); err != nil {
		return nil, err
	}
	if next, err = dsfs.LoadDataset(ctx, fsi.repo.Store(), ref.Path); err != nil {
		return nil, err
	}

	prevPath := next.PreviousPath
	if prevPath == "" {
		prev = &dataset.Dataset{}
	} else {
		// If the prior dataset can't be resolved quickly, we'll assume it isn't local
		// and time out the load
		// TODO (b5) - the proper way to handle this is to use a local dataset store
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Millisecond*700)
		defer cancel()

		if prev, err = dsfs.LoadDataset(timeoutCtx, fsi.repo.Store(), prevPath); err != nil {
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

	fileMap := make(map[string]FileStat)
	if next.Meta != nil {
		fileMap["meta"] = FileStat{Path: "meta"}
	}
	if next.BodyPath != "" || next.BodyFile() != nil {
		fileMap["body"] = FileStat{Path: "body"}
	}
	if next.Structure != nil && next.Structure.Schema != nil {
		fileMap["schema"] = FileStat{Path: "schema"}
	}
	return fsi.CalculateStateTransition(ctx, prev, next, fileMap, nil)
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
