package fsi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

// InitParams encapsulates parameters for fsi.InitDataset
type InitParams struct {
	Dir            string
	Name           string
	Format         string
	Mkdir          string
	SourceBodyPath string
}

func concatFunc(f1, f2 func()) func() {
	return func() {
		f1()
		f2()
	}
}

// PrepareToWrite is called before init writes the components to the filesystem. Used by tests.
var PrepareToWrite = func(comp component.Component) {
	// hook
}

// InitDataset creates a new dataset
func (fsi *FSI) InitDataset(p InitParams) (name string, err error) {
	// Create a rollback handler
	rollback := func() {
		log.Debug("did rollback InitDataset due to error")
	}
	defer func() {
		if rollback != nil {
			log.Debug("InitDataset rolling back...")
			rollback()
		}
	}()

	if !dsref.IsValidName(p.Name) {
		return "", dsref.ErrDescribeValidName
	}
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
		err := os.Mkdir(targetPath, os.ModePerm)
		if err != nil {
			if strings.Contains(err.Error(), "file exists") {
				// Not an error if directory already exists
				err = nil
			} else {
				return "", err
			}
		} else {
			// If directory was successfully created, add a step to the rollback in case future
			// steps fail.
			rollback = concatFunc(
				func() {
					log.Debugf("removing directory %q during rollback", targetPath)
					if err := os.Remove(targetPath); err != nil {
						log.Debugf("error while removing directory %q: %s", targetPath, err)
					}
				}, rollback)
		}
	}

	if err = canInitDir(targetPath); err != nil {
		return "", err
	}

	ref := &reporef.DatasetRef{Peername: "me", Name: p.Name}

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
		return "", fmt.Errorf(`invalid format "%s", only "csv" and "json" accepted`, p.Format)
	}

	// Create the link file, containing the dataset reference.
	var undo func()
	if name, undo, err = fsi.CreateLink(targetPath, ref.AliasString()); err != nil {
		return name, err
	}
	// If future steps fail, rollback the link creation.
	rollback = concatFunc(undo, rollback)

	// Construct the dataset to write to the working directory
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
			log.Errorf("opening entry reader: %s", err)
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
		bodySchema = dataset.BaseSchemaObject
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
	PrepareToWrite(container)
	for _, compName := range component.AllSubcomponentNames() {
		aComp := container.Base().GetSubcomponent(compName)
		if aComp != nil {
			wroteFile, err := aComp.WriteTo(targetPath)
			if err != nil {
				log.Errorf("writing component file %s: %s", compName, err)
				return "", err
			}
			// If future steps fail, rollback the components that have been written
			rollback = concatFunc(func() {
				if wroteFile != "" {
					log.Debugf("removing file %q during rollback", wroteFile)
					if err := os.Remove(wroteFile); err != nil {
						log.Debugf("error while removing file %q: %s", wroteFile, err)
					}
				}
			}, rollback)
		}
	}

	if err = fsi.repo.Logbook().WriteDatasetInit(context.TODO(), ref.Name); err != nil {
		if err == logbook.ErrNoLogbook {
			rollback = func() {}
			return name, nil
		}
		return name, err
	}

	// Success, no need to rollback.
	rollback = nil
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
