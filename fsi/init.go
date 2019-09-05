package fsi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/qri/repo"
)

// InitParams encapsulates parameters for fsi.InitDataset
type InitParams struct {
	Dir            string
	Name           string
	Format         string
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

	if err = canInitDir(p.Dir); err != nil {
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
	if name, err = fsi.CreateLink(p.Dir, ref.AliasString()); err != nil {
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
	if err := ioutil.WriteFile(filepath.Join(p.Dir, "meta.json"), metaSkeleton, os.ModePerm); err != nil {
		return name, err
	}

	var (
		schema map[string]interface{}
		data   []byte
	)
	if p.Format == "csv" {
		schema = map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "array",
				"items": []interface{}{
					// First column
					map[string]interface{}{
						"type":  "string",
						"title": "name",
					},
					// Second column
					map[string]interface{}{
						"type":  "string",
						"title": "describe",
					},
					// Third column
					map[string]interface{}{
						"type":  "integer",
						"title": "quantity",
					},
				},
			},
		}
	} else {
		schema = map[string]interface{}{
			"type": "object",
		}
	}
	data, err = json.MarshalIndent(schema, "", " ")
	if err := ioutil.WriteFile(filepath.Join(p.Dir, "schema.json"), data, os.ModePerm); err != nil {
		return name, err
	}

	var bodyBytes []byte
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
	}
	bodyFilename := filepath.Join(p.Dir, fmt.Sprintf("body.%s", p.Format))
	if err := ioutil.WriteFile(bodyFilename, bodyBytes, os.ModePerm); err != nil {
		return "", err
	}

	return name, err
}

func canInitDir(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, QriRefFilename)); !os.IsNotExist(err) {
		return fmt.Errorf("working directory is already linked, .qri-ref exists")
	}
	if _, err := os.Stat(filepath.Join(dir, "meta.json")); !os.IsNotExist(err) {
		// TODO(dlong): Instead, import the meta.json file for the new dataset
		return fmt.Errorf("cannot initialize new dataset, meta.json exists")
	}
	if _, err := os.Stat(filepath.Join(dir, "schema.json")); !os.IsNotExist(err) {
		// TODO(dlong): Instead, import the schema.json file for the new dataset
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
