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
	qerr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/fsi/linkfile"
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
	UseDscache     bool
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
func (fsi *FSI) InitDataset(p InitParams) (ref dsref.Ref, err error) {
	ctx := context.TODO()

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
		return ref, dsref.ErrDescribeValidName
	}
	if p.Dir == "" {
		return ref, fmt.Errorf("directory is required to initialize a dataset")
	}

	if fi, err := os.Stat(p.Dir); err != nil {
		return ref, err
	} else if !fi.IsDir() {
		return ref, fmt.Errorf("invalid path to initialize. '%s' is not a directory", p.Dir)
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
				return ref, err
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

	// Make sure we're not going to overwrite any files in the directory being initialized.
	// Pass the sourceBodyPath, because it's okay if this file already exists, as long as its
	// being used to create the body.
	if err = fsi.CanInitDatasetWorkDir(targetPath, p.SourceBodyPath); err != nil {
		return ref, err
	}

	book := fsi.repo.Logbook()

	// TODO(b5) - not a fan of relying on logbook for the current username
	ref = dsref.Ref{Username: book.Username(), Name: p.Name}
	// Validate dataset name. The `init` command must only be used for creating new datasets.
	// Make sure a dataset with this name does not exist in your repo.
	if _, err = fsi.repo.ResolveRef(ctx, &ref); err == nil {
		err = fmt.Errorf("a dataset named %q already exists", ref.Human())
		msg := `can't init a dataset with a name that already exists, use the checkout command
to create a working directory for an existing dataset`
		return ref, qerr.New(err, msg)
	}

	// write an InitID
	ref.InitID, err = fsi.repo.Logbook().WriteDatasetInit(ctx, ref.Name)
	if err != nil {
		if err == logbook.ErrNoLogbook {
			rollback = func() {}
			return ref, nil
		}
		return ref, err
	}

	rollback = concatFunc(
		func() {
			log.Debugf("removing log from logbook %q", ref)
			if err := book.RemoveLog(ctx, ref); err != nil {
				log.Error(err)
			}
		}, rollback)

	// Derive format from --source-body-path if provided.
	if p.Format == "" && p.SourceBodyPath != "" {
		ext := filepath.Ext(p.SourceBodyPath)
		if len(ext) > 0 {
			p.Format = ext[1:]
		}
	}

	// Validate dataset format
	if p.Format != "csv" && p.Format != "json" {
		return ref, fmt.Errorf(`invalid format "%s", only "csv" and "json" accepted`, p.Format)
	}

	// add the versionInfo to the refstore so CreateLink has something to read from
	// TODO (b5) - this probably means we shouldn't be using CreateLink here,
	// but I'd like to have one place that fires LinkCreated Events, maybe a private function
	// that both the exported CreateLink & Init can call
	vi := ref.VersionInfo()
	vi.FSIPath = targetPath
	if err := repo.PutVersionInfoShim(fsi.repo, &vi); err != nil {
		return ref, err
	}

	// Create the link file, containing the dataset reference.
	var undo func()
	if _, undo, err = fsi.CreateLink(targetPath, ref); err != nil {
		return ref, err
	}
	// If future steps fail, rollback the link creation.
	rollback = concatFunc(undo, rollback)

	// Construct the dataset to write to the working directory
	initDs := &dataset.Dataset{
		// Add an empty meta.
		Meta: &dataset.Meta{Title: ""},
	}

	// Add body file.
	var bodySchema map[string]interface{}
	if p.SourceBodyPath != "" {
		initDs.BodyPath = p.SourceBodyPath
		// Create structure by detecting it from the body.
		file, err := os.Open(p.SourceBodyPath)
		if err != nil {
			return ref, err
		}
		defer file.Close()
		// TODO(dlong): This should move into `dsio` package.
		entries, err := component.OpenEntryReader(file, p.Format)
		if err != nil {
			log.Errorf("opening entry reader: %s", err)
			return ref, err
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
				return ref, err
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

	// Success, no need to rollback.
	rollback = nil
	return ref, nil
}

// CanInitDatasetWorkDir returns nil if the directory can init a dataset, or an error if not
func (fsi *FSI) CanInitDatasetWorkDir(dir, sourceBodyPath string) error {
	// Get the source-body-path relative to the directory that we're initializing.
	relBodyPath := sourceBodyPath
	if strings.HasPrefix(relBodyPath, dir) {
		relBodyPath = strings.TrimPrefix(relBodyPath, dir)
		// Removing the directory may leave a leading slash, if the dirname did not end in a slash.
		if strings.HasPrefix(relBodyPath, "/") {
			relBodyPath = strings.TrimPrefix(relBodyPath, "/")
		}
	}

	// Check if .qri-ref link file already exists.
	if linkfile.ExistsInDir(dir) {
		return fmt.Errorf("working directory is already linked, .qri-ref exists")
	}
	// Check if other component files exist. If sourceBodyPath is provided, it's not an error
	// if its filename exists.
	if _, err := os.Stat(filepath.Join(dir, "meta.json")); !os.IsNotExist(err) {
		return fmt.Errorf("cannot initialize new dataset, meta.json exists")
	}
	if _, err := os.Stat(filepath.Join(dir, "structure.json")); !os.IsNotExist(err) {
		return fmt.Errorf("cannot initialize new dataset, structure.json exists")
	}
	if _, err := os.Stat(filepath.Join(dir, "body.csv")); !os.IsNotExist(err) {
		if relBodyPath != "body.csv" {
			return fmt.Errorf("cannot initialize new dataset, body.csv exists")
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "body.json")); !os.IsNotExist(err) {
		if relBodyPath != "body.json" {
			return fmt.Errorf("cannot initialize new dataset, body.json exists")
		}
	}

	return nil
}
