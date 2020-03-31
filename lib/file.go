package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/archive"
	"github.com/qri-io/qri/base/fill"
	"gopkg.in/yaml.v2"
)

// ReadDatasetFiles reads zero or more files, each representing a dataset or component of a
// dataset, and deserializes them, merging the results into a single dataset object. It is an
// error to provide any combination of files whose contents overlap (modify the same component).
func ReadDatasetFiles(pathList ...string) (*dataset.Dataset, error) {
	// If there's only a single file provided, read it and return the dataset.
	if len(pathList) == 1 {
		ds, _, err := readSingleFile(pathList[0])
		return ds, err
	}

	// If there's multiple files provided, read each one and merge them. Any exclusive
	// component is an error, any component showing up multiple times is an error.
	foundKinds := make(map[string]bool)
	ds := dataset.Dataset{}
	for _, p := range pathList {
		component, kind, err := readSingleFile(p)
		if err != nil {
			return nil, err
		}

		if kind == "zip" || kind == "ds" {
			return nil, fmt.Errorf("conflict, cannot save a full dataset with other components")
		}
		if _, ok := foundKinds[kind]; ok {
			return nil, fmt.Errorf("conflict, multiple components of kind \"%s\"", kind)
		}
		foundKinds[kind] = true

		ds.Assign(component)
	}

	return &ds, nil
}

// readSingleFile reads a single file, either a full dataset or component, and returns it as
// a dataset and a string specifying the kind of component that was created
func readSingleFile(path string) (*dataset.Dataset, string, error) {
	ds := dataset.Dataset{}
	switch qfs.PathKind(path) {
	case "http":
		// currently the only supported type of file url is a zip archive
		resp, err := http.Get(path)
		if err != nil {
			return nil, "", err
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, "", err
		}
		resp.Body.Close()
		err = archive.UnzipDatasetBytes(data, &ds)
		return &ds, "zip", nil

	case "ipfs":
		return nil, "", fmt.Errorf("reading dataset files from IPFS currently unsupported")

	case "local":
		f, err := os.Open(path)
		if err != nil {
			return nil, "", err
		}

		fileExt := strings.ToLower(filepath.Ext(path))
		switch fileExt {
		case ".yaml", ".yml":
			data, err := ioutil.ReadAll(f)
			if err != nil {
				return nil, "", err
			}

			fields := make(map[string]interface{})
			if err = yaml.Unmarshal(data, fields); err != nil {
				return nil, "", err
			}

			kind, err := fillDatasetOrComponent(fields, path, &ds)
			return &ds, kind, err

		case ".json":
			fields := make(map[string]interface{})
			if err = json.NewDecoder(f).Decode(&fields); err != nil {
				if strings.HasPrefix(err.Error(), "json: cannot unmarshal array") {
					err = fmt.Errorf("json has top-level type \"array\", cannot be a dataset file")
				}
				return nil, "", err
			}
			kind, err := fillDatasetOrComponent(fields, path, &ds)
			return &ds, kind, err

		case ".zip":
			data, err := ioutil.ReadAll(f)
			if err != nil {
				return nil, "", err
			}
			err = archive.UnzipDatasetBytes(data, &ds)
			return &ds, "zip", err

		case ".star":
			// starlark files are assumed to be a transform script with no additional
			// tranform component details:
			ds.Transform = &dataset.Transform{ScriptPath: path}
			ds.Transform.SetScriptFile(qfs.NewMemfileReader("transform.star", f))
			return &ds, "tf", nil

		case ".html":
			// html files are assumped to be a viz script with no additional viz
			// component details
			// TODO(dlong): Deprecate viz, assume "html" is a readme
			ds.Viz = &dataset.Viz{ScriptPath: path}
			ds.Viz.Format = "html"
			ds.Viz.SetScriptFile(qfs.NewMemfileReader("viz.html", f))
			return &ds, "vz", nil

		case ".md":
			// md files are assumped to be a readme file
			ds.Readme = &dataset.Readme{ScriptPath: path}
			ds.Readme.Format = "md"
			ds.Readme.SetScriptFile(qfs.NewMemfileReader("readme.md", f))
			return &ds, "rm", nil

		default:
			return nil, "", fmt.Errorf("error, unrecognized file extension: \"%s\"", fileExt)
		}
	default:
		return nil, "", fmt.Errorf("error, unknown path kind: \"%s\"", qfs.PathKind(path))
	}
}

func fillDatasetOrComponent(fields map[string]interface{}, path string, ds *dataset.Dataset) (string, error) {
	var target interface{}
	target = ds
	kind := "ds"

	if kindStr, ok := fields["qri"].(string); ok && len(kindStr) >= 2 {
		switch kindStr[:2] {
		case "rm":
			ds.Readme = &dataset.Readme{}
			target = ds.Readme
			kind = "rm"
		case "md":
			ds.Meta = &dataset.Meta{}
			target = ds.Meta
			kind = "md"
		case "cm":
			ds.Commit = &dataset.Commit{}
			target = ds.Commit
			kind = "cm"
		case "st":
			ds.Structure = &dataset.Structure{}
			target = ds.Structure
			kind = "st"
		}
	}

	if err := fill.Struct(fields, target); err != nil {
		return "", err
	}
	absDatasetPaths(path, ds)
	return kind, nil
}

// absDatasetPaths converts any relative filepath references in a Dataset to
// their absolute counterpart
func absDatasetPaths(path string, dsp *dataset.Dataset) {
	base := filepath.Dir(path)
	if dsp.BodyPath != "" && qfs.PathKind(dsp.BodyPath) == "local" && !filepath.IsAbs(dsp.BodyPath) {
		dsp.BodyPath = filepath.Join(base, dsp.BodyPath)
	}
	if dsp.Transform != nil && qfs.PathKind(dsp.Transform.ScriptPath) == "local" && !filepath.IsAbs(dsp.Transform.ScriptPath) {
		dsp.Transform.ScriptPath = filepath.Join(base, dsp.Transform.ScriptPath)
	}
	if dsp.Viz != nil && qfs.PathKind(dsp.Viz.ScriptPath) == "local" && !filepath.IsAbs(dsp.Viz.ScriptPath) {
		dsp.Viz.ScriptPath = filepath.Join(base, dsp.Viz.ScriptPath)
	}
	if dsp.Readme != nil && qfs.PathKind(dsp.Readme.ScriptPath) == "local" && !filepath.IsAbs(dsp.Readme.ScriptPath) {
		dsp.Readme.ScriptPath = filepath.Join(base, dsp.Readme.ScriptPath)
	}
}
