package fsi

import (
	"fmt"
	"os"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/component"
)

// ReadDir reads the component files in the directory, and returns a dataset
func ReadDir(dir string) (*dataset.Dataset, error) {
	components, err := component.ListDirectoryComponents(dir)
	if err != nil {
		return nil, err
	}
	err = component.ExpandListedComponents(components, nil)
	if err != nil {
		return nil, err
	}
	problems := GetProblems(components)
	if problems != "" {
		return nil, fmt.Errorf(problems)
	}
	ds, err := component.ToDataset(components)
	if err != nil {
		return nil, err
	}
	return ds, nil
}

// GetProblems returns the problem messages on a component collection
func GetProblems(comp component.Component) string {
	fcomp, ok := comp.(*component.FilesysComponent)
	if !ok {
		return ""
	}

	m := fcomp.BaseComponent.Subcomponents

	problems := ""
	for key := range m {
		comp := m[key].Base()
		if comp.ProblemKind != "" {
			if problems != "" {
				problems = fmt.Sprintf("%s ", problems)
			}
			problems = fmt.Sprintf("%s%s:[%s]", problems, comp.ProblemKind, comp.ProblemMessage)
		}
	}
	return problems
}

// WriteComponents writes components of the dataset to the given path, as individual files.
func WriteComponents(ds *dataset.Dataset, dirPath string, resolver qfs.Filesystem) error {
	// TODO(dlong): In the future, use ListDirectoryComponents(dirPath) to figure out what
	// files exist, project this component.Component onto those files. This will handle
	// things like writing a meta component into dataset.json's meta component instead of
	// a conflicting meta.json
	comp := component.ConvertDatasetToComponents(ds, resolver)
	comp.Base().RemoveSubcomponent("commit")
	comp.DropDerivedValues()

	for _, compName := range component.AllSubcomponentNames() {
		aComp := comp.Base().GetSubcomponent(compName)
		if aComp != nil {
			aComp.WriteTo(dirPath)
		}
	}

	return nil
}

// WriteComponent writes the component with the given name to the directory
func WriteComponent(comp component.Component, name string, dirPath string) error {
	aComp := comp.Base().GetSubcomponent(name)
	if aComp == nil {
		return nil
	}
	return aComp.WriteTo(dirPath)
}

// DeleteComponent deletes the component with the given name from the directory
func DeleteComponent(comp component.Component, name string, dirPath string) error {
	aComp := comp.Base().GetSubcomponent(name)
	if aComp == nil {
		return nil
	}
	return aComp.RemoveFrom(dirPath)
}

// DeleteDatasetFiles removes mapped files from a directory. if the result of
// deleting all files leaves the directory empty, it will remove the directory
// as well
func DeleteDatasetFiles(dirPath string) error {
	components, err := component.ListDirectoryComponents(dirPath)
	if err != nil {
		return err
	}

	allComponents := append(component.AllSubcomponentNames(), "dataset")

	for _, compName := range allComponents {
		subc := components.Base().GetSubcomponent(compName)
		if subc == nil {
			continue
		}
		// ignore not found errors. multiple components can be specified in the
		// same dataset file, creating multiple remove attempts to the same path
		if err := os.Remove(subc.Base().SourceFile); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	// attempt to remove the directory, ignoring "directory not empty" errors
	if err := os.Remove(dirPath); err != nil && !strings.Contains(err.Error(), "directory not empty") {
		return err
	}

	return nil
}
