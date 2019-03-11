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
	"github.com/qri-io/qri/base"
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

// ReadDatasetFile decodes a dataset document into a Dataset
func ReadDatasetFile(path string) (ds *dataset.Dataset, err error) {
	var (
		resp *http.Response
		f    *os.File
		data []byte
	)

	ds = &dataset.Dataset{}

	switch pathKind(path) {
	case "http":
		// currently the only supported type of file url is a zip archive
		resp, err = http.Get(path)
		if err != nil {
			return
		}
		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		resp.Body.Close()
		err = dsutil.UnzipDatasetBytes(data, ds)
		return

	case "ipfs":
		return nil, fmt.Errorf("reading dataset files from IPFS currently unsupported")

	case "file":
		f, err = os.Open(path)
		if err != nil {
			return
		}

		fileExt := strings.ToLower(filepath.Ext(path))
		switch fileExt {
		case ".yaml", ".yml":
			data, err = ioutil.ReadAll(f)
			if err != nil {
				return
			}

			fields := make(map[string]interface{})
			if err = yaml.Unmarshal(data, fields); err != nil {
				return
			}
			err = fillDatasetOrComponent(fields, path, ds)

		case ".json":
			fields := make(map[string]interface{})
			if err = json.NewDecoder(f).Decode(&fields); err != nil {
				return
			}
			err = fillDatasetOrComponent(fields, path, ds)

		case ".zip":
			data, err = ioutil.ReadAll(f)
			if err != nil {
				return
			}
			err = dsutil.UnzipDatasetBytes(data, ds)
			return

		case ".star":
			// starlark files are assumed to be a transform script with no additional
			// tranform component details:
			ds.Transform = &dataset.Transform{ScriptPath: path}
			ds.Transform.SetScriptFile(qfs.NewMemfileReader("transform.star", f))
			return

		case ".html":
			// html files are assumped to be a viz script with no additional viz
			// component details
			ds.Viz = &dataset.Viz{ScriptPath: path}
			ds.Viz.SetScriptFile(qfs.NewMemfileReader("viz.html", f))
			return

		default:
			return nil, fmt.Errorf("error, unrecognized file extension: \"%s\"", fileExt)
		}
	}
	return
}

func fillDatasetOrComponent(fields map[string]interface{}, path string, ds *dataset.Dataset) (err error) {
	var fill interface{}
	fill = ds

	if kindStr, ok := fields["qri"].(string); ok && len(kindStr) > 3 {
		switch kindStr[:2] {
		case "md":
			ds.Meta = &dataset.Meta{}
			fill = ds.Meta
		case "cm":
			ds.Commit = &dataset.Commit{}
			fill = ds.Commit
		case "st":
			ds.Structure = &dataset.Structure{}
			fill = ds.Structure
		}
	}

	if err = base.FillStruct(fields, fill); err != nil {
		return err
	}
	absDatasetPaths(path, ds)
	return nil
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
