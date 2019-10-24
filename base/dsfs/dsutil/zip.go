package dsutil

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base/dsfs"
)

// WriteZipArchive generates a zip archive of a dataset and writes it to w
func WriteZipArchive(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset, format string, ref string, w io.Writer) error {
	zw := zip.NewWriter(w)

	// Dataset header, contains meta, structure, and commit
	dsf, err := zw.Create(fmt.Sprintf("dataset.%s", format))
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	var dsdata []byte
	switch format {
	case "json":
		dsdata, err = json.MarshalIndent(ds, "", "  ")
		if err != nil {
			return err
		}
	case "yaml":
		dsdata, err = yaml.Marshal(ds)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown file format: \"%s\"", format)
	}

	_, err = dsf.Write(dsdata)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	// Reference to dataset, as a string
	target, err := zw.Create("ref.txt")
	if err != nil {
		return err
	}
	_, err = io.WriteString(target, ref)
	if err != nil {
		return err
	}

	// Transform script
	if ds.Transform != nil && ds.Transform.ScriptPath != "" {
		script, err := store.Get(ctx, ds.Transform.ScriptPath)
		if err != nil {
			return err
		}
		target, err := zw.Create("transform.star")
		if err != nil {
			return err
		}
		_, err = io.Copy(target, script)
		if err != nil {
			return err
		}
	}

	// Viz template
	if ds.Viz != nil {
		if ds.Viz.ScriptPath != "" {
			script, err := store.Get(ctx, ds.Viz.ScriptPath)
			if err != nil {
				return err
			}
			target, err := zw.Create("viz.html")
			if err != nil {
				return err
			}
			_, err = io.Copy(target, script)
			if err != nil {
				return err
			}
		}

		if ds.Viz.RenderedPath != "" {
			// TODO (b5) - rendered viz isn't always being properly added to the
			// encoded DAG, causing this to hang indefinitely on a network lookup.
			// Use a short timeout for now to prevent the process from running too
			// long. We should come up with a more permanent fix for this.
			withTimeout, done := context.WithTimeout(ctx, time.Millisecond*250)
			defer done()
			rendered, err := store.Get(withTimeout, ds.Viz.RenderedPath)
			if err != nil {
				return err
			}
			target, err := zw.Create("index.html")
			if err != nil {
				return err
			}
			_, err = io.Copy(target, rendered)
			if err != nil {
				return err
			}
		}
	}

	// Body
	datadst, err := zw.Create(fmt.Sprintf("body.%s", ds.Structure.Format))
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	datasrc, err := dsfs.LoadBody(ctx, store, ds)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	if _, err = io.Copy(datadst, datasrc); err != nil {
		log.Debug(err.Error())
		return err
	}
	return zw.Close()
}

// UnzipDatasetBytes is a convenince wrapper for UnzipDataset
func UnzipDatasetBytes(zipData []byte, ds *dataset.Dataset) error {
	return UnzipDataset(bytes.NewReader(zipData), int64(len(zipData)), ds)
}

// UnzipDataset reads a zip file from a filename and returns a full dataset with components
func UnzipDataset(r io.ReaderAt, size int64, ds *dataset.Dataset) error {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return err
	}

	contents, err := unzipGetContents(zr)
	if err != nil {
		return err
	}

	fileData, ok := contents["dataset.json"]
	if !ok {
		return fmt.Errorf("no dataset.json found in the provided zip")
	}
	if err = json.Unmarshal(fileData, ds); err != nil {
		return err
	}

	// TODO - do a smarter iteration for body format
	if bodyData, ok := contents["body.json"]; ok {
		ds.BodyBytes = bodyData
		ds.BodyPath = ""
	}
	if bodyData, ok := contents["body.csv"]; ok {
		ds.BodyBytes = bodyData
		ds.BodyPath = ""
	}
	if bodyData, ok := contents["body.cbor"]; ok {
		ds.BodyBytes = bodyData
		ds.BodyPath = ""
	}

	if tfScriptData, ok := contents["transform.star"]; ok {
		if ds.Transform == nil {
			ds.Transform = &dataset.Transform{}
		}
		ds.Transform.ScriptBytes = tfScriptData
		ds.Transform.ScriptPath = ""
	}

	if vizScriptData, ok := contents["viz.html"]; ok {
		if ds.Viz == nil {
			ds.Viz = &dataset.Viz{}
		}
		ds.Viz.ScriptBytes = vizScriptData
		ds.Viz.ScriptPath = ""
	}

	// Get ref to existing dataset
	if refText, ok := contents["ref.txt"]; ok {
		refStr := string(refText)
		atPos := strings.Index(refStr, "@")
		if atPos == -1 {
			return fmt.Errorf("invalid dataset ref: no '@' found")
		}
		// Get name and peername
		datasetName := refStr[:atPos]
		sepPos := strings.Index(datasetName, "/")
		if sepPos == -1 {
			return fmt.Errorf("invalid dataset name: no '/' found")
		}
		ds.Peername = datasetName[:sepPos]
		ds.Name = datasetName[sepPos+1:]
	}
	return nil
}

// UnzipGetContents is a generic zip-unpack to a map of filename: contents
// with contents represented as strings
func UnzipGetContents(data []byte) (map[string]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	contents, err := unzipGetContents(zr)
	if err != nil {
		return nil, err
	}

	res := map[string]string{}
	for k, val := range contents {
		res[k] = string(val)
	}
	return res, nil
}

// unzipGetContents reads a zip file's contents and returns a map from filename to file data
func unzipGetContents(zr *zip.Reader) (map[string][]byte, error) {
	// Create a map from filenames in the zip to their json encoded contents.
	contents := make(map[string][]byte)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, err
		}
		contents[f.Name] = data
	}
	return contents, nil
}
