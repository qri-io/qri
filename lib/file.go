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
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/fill"
	"gopkg.in/yaml.v2"
)

// AbsPath adjusts the provided string to a path lib functions can work with
// because paths for Qri can come from the local filesystem, an http url, or
// the distributed web, Absolutizing is a little tricky
//
// If lib in put params call for a path, running input through AbsPath before
// calling a lib function should help reduce errors. calling AbsPath on empty
// string has no effect
func AbsPath(path *string) (err error) {
	if *path == "" {
		return
	}

	*path = strings.TrimSpace(*path)
	p := *path

	// bail on urls and ipfs hashes
	pk := pathKind(p)
	if pk == "http" || pk == "ipfs" {
		return
	}

	// TODO - perform tilda (~) expansion
	if filepath.IsAbs(p) {
		return
	}
	*path, err = filepath.Abs(p)
	return
}

func pathKind(path string) string {
	if path == "" {
		return "none"
	} else if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return "http"
	} else if strings.HasPrefix(path, "/ipfs") {
		return "ipfs"
	} else if strings.HasPrefix(path, "/map") || strings.HasPrefix(path, "/cafs") {
		return "cafs"
	}
	return "file"
}

// ReadDatasetFiles decodes a dataset document into a Dataset
func ReadDatasetFiles(pathList []string) (*dataset.Dataset, error) {
	// If there's only a single file provided, read it and return the dataset.
	if len(pathList) == 1 {
		ds, _, err := ReadSingleFile(pathList[0])
		return ds, err
	}

	// If there's multiple files provided, read each one and merge them. Any exclusive
	// component is an error, any component showing up multiple times is an error.
	foundKinds := make(map[string]bool)
	ds := dataset.Dataset{}
	for _, p := range pathList {
		component, kind, err := ReadSingleFile(p)
		if err != nil {
			return nil, err
		}

		if kind == "zip" || kind == "ds" {
			return nil, fmt.Errorf("")
		}
		if _, ok := foundKinds[kind]; ok {
			return nil, fmt.Errorf("conflict, multiple components of kind %s", kind)
		}
		foundKinds[kind] = true

		ds.Assign(component)
	}

	return &ds, nil
}

func ReadSingleFile(path string) (*dataset.Dataset, string, error) {
	ds := dataset.Dataset{}
	switch pathKind(path) {
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
		err = dsutil.UnzipDatasetBytes(data, &ds)
		return &ds, "zip", nil

	case "ipfs":
		return nil, "", fmt.Errorf("reading dataset files from IPFS currently unsupported")

	case "file":
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

			// TODO (b5): temp hack to deal with terrible interaction with fill_struct,
			// we should find a more robust solution to this, enforcing the assumption that all
			// dataset documents use string keys
			if sti, ok := fields["structure"].(map[interface{}]interface{}); ok {
				fields["structure"] = toMapIface(sti)
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
			err = dsutil.UnzipDatasetBytes(data, &ds)
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
			ds.Viz = &dataset.Viz{ScriptPath: path}
			ds.Viz.SetScriptFile(qfs.NewMemfileReader("viz.html", f))
			return &ds, "vz", nil

		default:
			return nil, "", fmt.Errorf("error, unrecognized file extension: \"%s\"", fileExt)
		}
	default:
		return nil, "", fmt.Errorf("error, unknown path kind: \"%s\"", pathKind(path))
	}
}

func toMapIface(i map[interface{}]interface{}) map[string]interface{} {
	mapi := map[string]interface{}{}
	for ikey, val := range i {
		switch x := val.(type) {
		case map[interface{}]interface{}:
			val = toMapIface(x)
		case []interface{}:
			for i, v := range x {
				if mapi, ok := v.(map[interface{}]interface{}); ok {
					x[i] = toMapIface(mapi)
				}
			}
		}

		if key, ok := ikey.(string); ok {
			mapi[key] = val
		}
	}
	return mapi
}

func fillDatasetOrComponent(fields map[string]interface{}, path string, ds *dataset.Dataset) (string, error) {
	var target interface{}
	target = ds
	kind := "ds"

	if kindStr, ok := fields["qri"].(string); ok && len(kindStr) > 3 {
		switch kindStr[:2] {
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
	if dsp.BodyPath != "" && pathKind(dsp.BodyPath) == "file" && !filepath.IsAbs(dsp.BodyPath) {
		dsp.BodyPath = filepath.Join(base, dsp.BodyPath)
	}
	if dsp.Transform != nil && pathKind(dsp.Transform.ScriptPath) == "file" && !filepath.IsAbs(dsp.Transform.ScriptPath) {
		dsp.Transform.ScriptPath = filepath.Join(base, dsp.Transform.ScriptPath)
	}
	if dsp.Viz != nil && pathKind(dsp.Viz.ScriptPath) == "file" && !filepath.IsAbs(dsp.Viz.ScriptPath) {
		dsp.Viz.ScriptPath = filepath.Join(base, dsp.Viz.ScriptPath)
	}
}
