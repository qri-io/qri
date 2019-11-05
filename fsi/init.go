package fsi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
)

// InitParams encapsulates parameters for fsi.InitDataset
type InitParams struct {
	Dir            string
	Name           string
	Format         string
	Mkdir          string
	SourceBodyPath string
}

// InitDataset creates a new dataset
func (fsi *FSI) InitDataset(p InitParams) (name string, err error) {
	// TODO (ramfox): at each failure, we should ensure we clean up any
	// file or directory creation by calling either os.Remove(targetPath)
	// or fsi.Unlink(targetPath, ref.AliasString())
	if p.Dir == "" {
		return "", fmt.Errorf("directory is required to initialize a dataset")
	}

	if fi, err := os.Stat(p.Dir); err != nil {
		return "", err
	} else if !fi.IsDir() {
		return "", fmt.Errorf("invalid path to initialize. '%s' is not a directory", p.Dir)
	}

	// Either use an existing directory, or create one at the given directory.
	var targetPath string
	if p.Mkdir == "" {
		targetPath = p.Dir
	} else {
		targetPath = filepath.Join(p.Dir, p.Mkdir)
		// Create the directory. It is not an error for the directory to already exist, as long
		// as it is not already linked, which is checked below.
		if err := os.Mkdir(targetPath, os.ModePerm); err != nil {
			return "", err
		}
	}

	if err = canInitDir(targetPath); err != nil {
		return "", err
	}

	ref := &repo.DatasetRef{Peername: "me", Name: p.Name}

	// Validate dataset name. The `init` command must only be used for creating new datasets.
	// Make sure a dataset with this name does not exist in your repo.
	if err = repo.CanonicalizeDatasetRef(fsi.repo, ref); err == nil {
		// TODO(dlong): Tell user to use `checkout` if the dataset already exists in their repo?
		return "", fmt.Errorf("a dataset with the name %s already exists in your repo", ref)
	}

	// Derive format from --source-body-path if provided.
	if p.Format == "" && p.SourceBodyPath != "" {
		ext := filepath.Ext(p.SourceBodyPath)
		if len(ext) > 0 {
			p.Format = ext[1:]
		}
	}

	// Validate dataset format
	if p.Format != "csv" && p.Format != "json" {
		return "", fmt.Errorf("invalid format \"%s\", only \"csv\" and \"json\" accepted", p.Format)
	}

	// Create the link file, containing the dataset reference.
	if name, err = fsi.CreateLink(targetPath, ref.AliasString()); err != nil {
		return name, err
	}

	initDs := &dataset.Dataset{}

	// Add an empty meta.
	initDs.Meta = &dataset.Meta{Title: ""}

	// Add body file.
	var bodySchema map[string]interface{}
	if p.SourceBodyPath != "" {
		initDs.BodyPath = p.SourceBodyPath
		// Create structure by detecting it from the body.
		file, err := os.Open(p.SourceBodyPath)
		if err != nil {
			return "", err
		}
		defer file.Close()
		// TODO(dlong): This should move into `dsio` package.
		entries, err := component.OpenEntryReader(file, p.Format)
		if err != nil {
			return "", err
		}
		initDs.Structure = entries.Structure()
	} else if p.Format == "csv" {
		initDs.Body = []interface{}{
			[]interface{}{"one", "two", 3},
			[]interface{}{"four", "five", 6},
		}
	} else if p.Format == "json" {
		initDs.Body = map[string]interface{}{
			"key": "value",
		}
	}

	// Add structure.
	if initDs.Structure == nil {
		initDs.Structure = &dataset.Structure{
			Format: p.Format,
		}
		if bodySchema != nil {
			initDs.Structure.Schema = bodySchema
		}
	}

	// Write components of the dataset to the working directory.
	container := component.ConvertDatasetToComponents(initDs, fsi.repo.Filesystem())
	for _, compName := range component.AllSubcomponentNames() {
		aComp := container.Base().GetSubcomponent(compName)
		if aComp != nil {
			aComp.WriteTo(targetPath)
		}
	}

	if err = fsi.repo.Logbook().WriteDatasetInit(context.TODO(), name); err != nil {
		if err == logbook.ErrNoLogbook {
			err = nil
			return name, nil
		}
		return name, err
	}
	return name, nil
}

func canInitDir(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, QriRefFilename)); !os.IsNotExist(err) {
		return fmt.Errorf("working directory is already linked, .qri-ref exists")
	}
	if _, err := os.Stat(filepath.Join(dir, "meta.json")); !os.IsNotExist(err) {
		// TODO(dlong): Instead, import the meta.json file for the new dataset
		return fmt.Errorf("cannot initialize new dataset, meta.json exists")
	}
	if _, err := os.Stat(filepath.Join(dir, "structure.json")); !os.IsNotExist(err) {
		// TODO(dlong): Instead, import the structure.json file for the new dataset
		return fmt.Errorf("cannot initialize new dataset, schema.json exists")
	}
	if _, err := os.Stat(filepath.Join(dir, "body.csv")); !os.IsNotExist(err) {
		// TODO(dlong): Instead, import the body.csv file for the new dataset
		return fmt.Errorf("cannot initialize new dataset, body.csv exists")
	}
	if _, err := os.Stat(filepath.Join(dir, "body.json")); !os.IsNotExist(err) {
		// TODO(dlong): Instead, import the body.json file for the new dataset
		return fmt.Errorf("cannot initialize new dataset, body.json exists")
	}

	return nil
}
