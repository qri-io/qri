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
)

// AbsPath adjusts the provdide string to a path lib functions can work with
// because paths for Qri can come from the local filesystem, an http url, or
// the distributed web, Absolutizing is a little tricky
//
// If lib in put params call for a path, running it through AbsPath before
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
	if _, err = os.Stat(p); os.IsNotExist(err) {
		return fmt.Errorf(`open "%s": no such file or directory`, p)
	}
	if filepath.IsAbs(p) {
		return
	}
	*path, err = filepath.Abs(p)
	return
}

func pathKind(path string) string {
	if path == "" {
		return "none"
	}
	if strings.HasPrefix(path, "http") {
		return "http"
	}
	if strings.HasPrefix(path, "/ipfs") {
		return "ipfs"
	}
	return "file"
}

// ReadDatasetFile decodes a dataset document into a DatasetPod
func ReadDatasetFile(path string) (dsp *dataset.DatasetPod, err error) {
	var (
		resp *http.Response
		f    *os.File
		data []byte
	)

	dsp = &dataset.DatasetPod{}

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
		err = dsutil.UnzipDatasetBytes(data, dsp)
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
			if err = dsutil.UnmarshalYAMLDatasetPod(data, dsp); err != nil {
				return
			}
			absDatasetPaths(path, dsp)

		case ".json":
			if err = json.NewDecoder(f).Decode(dsp); err != nil {
				return
			}
			absDatasetPaths(path, dsp)

		case ".zip":
			data, err = ioutil.ReadAll(f)
			if err != nil {
				return
			}
			err = dsutil.UnzipDatasetBytes(data, dsp)
			return

		default:
			return nil, fmt.Errorf("error, unrecognized file extension: \"%s\"", fileExt)
		}
	}
	return
}

// absDatasetPaths converts any relative filepath references in a DatasetPod to
// their absolute counterpart
func absDatasetPaths(path string, dsp *dataset.DatasetPod) {
	base := filepath.Base(path)
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
