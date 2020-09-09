package component

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/fill"
)

var (
	// ErrNoDatasetFiles indicates no data
	ErrNoDatasetFiles = fmt.Errorf("no dataset files provided")
)

// ListDirectoryComponents lists the relevant files and reads them into a component collection
// object. The resulting object has stat'ed each file, and has their mtimes, but no files
// have been read from disk. Conflicting files (such as both a "body.csv" and "body.json") will
// cause the "ProblemKind" and "ProblemMessage" fields to be set. Other conflicts may also exist,
// such as "meta" being in both "dataset.json" and "meta.json", but this function does not detect
// these kinds of problems because it does not read any files.
func ListDirectoryComponents(dir string) (Component, error) {
	knownFilenames := GetKnownFilenames()
	topLevel := FilesysComponent{}

	finfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	// Note that this traversal will be in a non-deterministic order, so nothing in this loop
	// should depend on list order.
	for _, fi := range finfos {
		ext := filepath.Ext(fi.Name())
		componentName := strings.ToLower(strings.TrimSuffix(fi.Name(), ext))
		allowedExtensions, ok := knownFilenames[componentName]
		if !ok {
			// If a file in this directory is not a known filename, ignore it
			continue
		}
		if !sliceContains(allowedExtensions, ext) {
			// Also ignore the file if it has an unknown file extension
			continue
		}
		absPath, _ := filepath.Abs(filepath.Join(dir, fi.Name()))
		// Check for conflict between this file and those already observed
		if holder := topLevel.GetSubcomponent(componentName); holder != nil {
			elem := holder.Base()
			elem.ProblemKind = "conflict"
			// Collect a message containing the paths of conflicting files
			msg := elem.ProblemMessage
			if msg == "" {
				msg = filepath.Base(elem.SourceFile)
			}
			// Sort the problem files so that the message is deterministic
			conflictFiles := append(strings.Split(msg, " "), filepath.Base(absPath))
			sort.Strings(conflictFiles)
			elem.ProblemMessage = strings.Join(conflictFiles, " ")
			continue
		}
		topLevel.SetSubcomponent(
			componentName,
			BaseComponent{
				ModTime:    fi.ModTime(),
				SourceFile: absPath,
				Format:     normalizeExtensionFormat(ext),
			},
		)
	}
	if topLevel.IsEmpty() {
		return nil, ErrNoDatasetFiles
	}
	return &topLevel, nil
}

// ExpandListedComponents will read whatever is necessary in order to discover all of the components
// that exist within this observation. For example, if a "dataset" exists, it will be read to find
// out if it contains a "meta", a "structure", etc. No other components are expanded, but this
// may change in the future if we decide another component can contain some other component. If
// the "dataset" file does not exist, an empty dataset component will be created.
func ExpandListedComponents(container Component, resolver qfs.Filesystem) error {
	filesysComponent, ok := container.(*FilesysComponent)
	if !ok {
		return fmt.Errorf("cannot expand non-filesys container")
	}

	ds := dataset.Dataset{}

	dsComponent := filesysComponent.GetSubcomponent("dataset")
	if dsComponent == nil {
		dsComponent = filesysComponent.SetSubcomponent("dataset", BaseComponent{})
	} else {
		fields, err := dsComponent.Base().LoadFile()
		if err != nil {
			// TODO(dlong): Better
			return err
		}

		if err := fill.Struct(fields, &ds); err != nil {
			// TODO(dlong): Fix me
			return err
		}
	}

	dsCont := dsComponent.(*DatasetComponent)
	dsCont.Value = &ds

	if ds.Commit != nil {
		comp := assignField(filesysComponent, "commit", dsComponent)
		if comp != nil {
			commit := comp.(*CommitComponent)
			commit.Value = ds.Commit
			commit.IsLoaded = true
		}
	}
	if ds.Meta != nil {
		comp := assignField(filesysComponent, "meta", dsComponent)
		if comp != nil {
			meta := comp.(*MetaComponent)
			meta.Value = ds.Meta
			meta.IsLoaded = true
		}
	}
	var bodyStructure *dataset.Structure
	if ds.Structure != nil {
		comp := assignField(filesysComponent, "structure", dsComponent)
		if comp != nil {
			structure := comp.(*StructureComponent)
			structure.Value = ds.Structure
			structure.IsLoaded = true
			bodyStructure = ds.Structure
		}
	}
	if ds.Readme != nil {
		comp := assignField(filesysComponent, "readme", dsComponent)
		if comp != nil {
			readme := comp.(*ReadmeComponent)
			readme.Resolver = resolver
			readme.Value = ds.Readme
			readme.IsLoaded = true
		}
	}
	if ds.Transform != nil {
		comp := assignField(filesysComponent, "transform", dsComponent)
		if comp != nil {
			readme := comp.(*TransformComponent)
			readme.Resolver = resolver
			readme.Value = ds.Transform
			readme.IsLoaded = true
		}
	}
	if ds.Body != nil {
		comp := assignField(filesysComponent, "body", dsComponent)
		if comp != nil {
			body := comp.(*BodyComponent)
			body.Resolver = resolver
			if bodyStructure != nil {
				body.Structure = bodyStructure
			}
		}
	}

	stComp := filesysComponent.GetSubcomponent("structure")
	bdComp := filesysComponent.GetSubcomponent("body")
	if stComp != nil && bdComp != nil {
		if structure, ok := stComp.(*StructureComponent); ok {
			if body, ok := bdComp.(*BodyComponent); ok {
				if structure.Value == nil || structure.Value.Schema == nil {
					structure.SchemaInference = func(ds *dataset.Dataset) (map[string]interface{}, error) {
						err := body.LoadAndFill(ds)
						if err != nil {
							return nil, err
						}
						return body.InferredSchema, nil
					}
				}
			}
		}
	}

	return nil
}

func assignField(target Component, componentName string, parent Component) Component {
	found := target.Base().GetSubcomponent(componentName)
	if found != nil {
		addFile := filepath.Base(parent.Base().SourceFile)
		existingFile := filepath.Base(found.Base().SourceFile)
		found.Base().ProblemKind = "conflict"
		found.Base().ProblemMessage = fmt.Sprintf("%s %s", existingFile, addFile)
		return nil
	}
	return target.Base().SetSubcomponent(
		componentName,
		BaseComponent{
			ModTime:    parent.Base().ModTime,
			SourceFile: parent.Base().SourceFile,
			Format:     parent.Base().Format,
		},
	)
}

// GetKnownFilenames returns a map containing all possible filenames (filebase and extension) for
// any file that can represent a component of a dataset.
func GetKnownFilenames() map[string][]string {
	componentExtensionTypes := []string{".json", ".yml", ".yaml"}
	bodyExtensionTypes := []string{".csv", ".json", ".cbor", ".xlsx"}
	readmeExtensionTypes := []string{".md", ".html"}
	return map[string][]string{
		"dataset":   componentExtensionTypes,
		"commit":    componentExtensionTypes,
		"meta":      componentExtensionTypes,
		"structure": componentExtensionTypes,
		// TODO(dlong): Viz is deprecated
		"viz":       []string{".html"},
		"readme":    readmeExtensionTypes,
		"transform": []string{".star"},
		"body":      bodyExtensionTypes,
	}
}

// IsKnownFilename returns whether the file is a known component filename.
func IsKnownFilename(fullpath string, known map[string][]string) bool {
	if known == nil {
		known = GetKnownFilenames()
	}
	basename := filepath.Base(fullpath)
	ext := filepath.Ext(basename)
	onlybase := strings.ToLower(basename[:len(basename)-len(ext)])
	allowedExtensions, ok := known[onlybase]
	if !ok {
		return false
	}
	for _, allow := range allowedExtensions {
		if allow == ext {
			return true
		}
	}
	return false
}

func normalizeExtensionFormat(text string) string {
	if strings.HasPrefix(text, ".") {
		text = text[1:]
	}
	if text == "yml" {
		text = "yaml"
	}
	return text
}

func sliceContains(subject []string, needle string) bool {
	for _, elem := range subject {
		if elem == needle {
			return true
		}
	}
	return false
}
