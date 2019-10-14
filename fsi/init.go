package fsi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/logbook"
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
	if p.Dir == "" {
		return "", fmt.Errorf("directory is required to initialize a dataset")
	}

	if fi, err := os.Stat(p.Dir); err != nil {
		return "", err
	} else if !fi.IsDir() {
		return "", fmt.Errorf("invalid path to initialize. '%s' is not a directory", p.Dir)
	}

	// TODO(dlong): This function should more closely resemble Checkout in lib/fsi.go

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

	// Create a skeleton meta.json file.
	metaSkeleton := []byte(`{
		"title": "",
		"description": "",
		"keywords": [],
		"homeURL": ""
	}
	`)
	if err := ioutil.WriteFile(filepath.Join(targetPath, "meta.json"), metaSkeleton, os.ModePerm); err != nil {
		return name, err
	}

	var bodyBytes []byte
	var schema map[string]interface{}
	if p.SourceBodyPath == "" {
		// Create a skeleton body file.
		if p.Format == "csv" {
			bodyBytes = []byte("one,two,3\nfour,five,6")
		} else if p.Format == "json" {
			bodyBytes = []byte(`{
  "key": "value"
}`)
		} else {
			return "", fmt.Errorf("unknown body format %s", p.Format)
		}
	} else {
		// Create body file by reading the sourcefile.
		if bodyBytes, err = ioutil.ReadFile(p.SourceBodyPath); err != nil {
			return "", err
		}
		file, err := os.Open(p.SourceBodyPath)
		if err != nil {
			return "", err
		}
		defer file.Close()
		entries, err := OpenEntryReader(file, p.Format)
		if err == nil {
			err = dsio.EachEntry(entries, func(int, dsio.Entry, error) error { return nil })
			if err != nil {
				schema = nil
			}
			schema = entries.Structure().Schema
		}
	}
	bodyFilename := filepath.Join(targetPath, fmt.Sprintf("body.%s", p.Format))
	if err := ioutil.WriteFile(bodyFilename, bodyBytes, os.ModePerm); err != nil {
		return "", err
	}

	// Create basic structure (no derived values) based on Format
	structureBytes, err := createBasicStructure(p.Format, schema)
	if err != nil {
		return "", err
	}
	// use format to determine basic formatConfig
	structureFilename := filepath.Join(targetPath, "structure.json")
	if err := ioutil.WriteFile(structureFilename, structureBytes, os.ModePerm); err != nil {
		return "", err
	}

	err = fsi.repo.Logbook().WriteNameInit(context.TODO(), name)
	if err == logbook.ErrNoLogbook {
		err = nil
	}
	return name, err
}

func createBasicStructure(format string, schema map[string]interface{}) ([]byte, error) {
	var err error
	formatConfigBytes := []byte{}
	schemaBytes := []byte{}
	switch format {
	case "csv":
		formatConfigBytes, err = json.Marshal(dataset.CSVOptions{})
		if err != nil {
			return nil, err
		}
	case "json":
		formatConfigBytes, err = json.Marshal(map[string]interface{}{"pretty": false})
		if err != nil {
			return nil, err
		}
	case "xlsx":
		formatConfigBytes, err = json.Marshal(dataset.XLSXOptions{SheetName: "sheet1"})
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown body format '%s'", format)
	}
	if len(schema) == 0 {
		schema = nil
	}
	schemaBytes, err = json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	structureStr := fmt.Sprintf(`{"format":"%s","formatConfig":%s,"schema":%s}`, format, formatConfigBytes, schemaBytes)
	return []byte(structureStr), nil
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
