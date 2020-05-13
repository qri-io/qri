package archive

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

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/fsi/linkfile"
)

// WriteZip generates a zip archive of a dataset and writes it to w
func WriteZip(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset, format, initID string, ref dsref.Ref, w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	st := ds.Structure

	if ref.Path == "" && ds.Path != "" {
		ref.Path = ds.Path
	}

	// Iterate the individual components of the dataset
	dsComp := component.ConvertDatasetToComponents(ds, store)
	for _, compName := range component.AllSubcomponentNames() {
		aComp := dsComp.Base().GetSubcomponent(compName)
		if aComp == nil {
			continue
		}

		data, err := aComp.StructuredData()
		if err != nil {
			log.Error("component %q, geting structured data: %s", compName, err)
			continue
		}

		// Specially serialize the body to a file in the zip
		if compName == "body" && st != nil {
			body, err := component.SerializeBody(data, st)
			if err != nil {
				log.Error("component %q, serializing body: %s", compName, err)
				continue
			}

			w, err := zw.Create(fmt.Sprintf("%s.%s", compName, st.Format))
			if err != nil {
				log.Error("component %q, creating zip writer: %s", compName, err)
				continue
			}

			w.Write(body)
			continue
		}

		// TODO(dustmop): The transform component outputs a json file, with a path string
		// to the transform script in IPFS. Consider if Components should have a
		// serialize method that gets the script for transform, and maybe the body contents,
		// but a json struct for everything else. Follow up in another PR.

		// For any other component, serialize it as json in the zip
		w, err := zw.Create(fmt.Sprintf("%s.json", compName))
		if err != nil {
			log.Error("component %q, creating zip writer: %s", compName, err)
			continue
		}

		text, err := json.MarshalIndent(data, "", " ")
		if err != nil {
			log.Error("component %q, marshalling data: %s", compName, err)
			continue
		}
		w.Write(text)
	}

	// Add a linkfile in the zip, which can be used to connect the dataset back to its history
	w, err := zw.Create(linkfile.RefLinkTextFilename)
	if err != nil {
		log.Error(err)
	} else {
		linkfile.WriteRef(w, ref)
	}

	return nil
}

// TODO (b5) - rendered viz isn't always being properly added to the
// encoded DAG, causing this to hang indefinitely on a network lookup.
// Use a short timeout for now to prevent the process from running too
// long. We should come up with a more permanent fix for this.
func maybeWriteRenderedViz(ctx context.Context, store cafs.Filestore, zw *zip.Writer, vizPath string) error {
	withTimeout, done := context.WithTimeout(ctx, time.Millisecond*250)
	defer done()
	rendered, err := store.Get(withTimeout, vizPath)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}

	target, err := zw.Create("index.html")
	if err != nil {
		return err
	}
	_, err = io.Copy(target, rendered)
	return err
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
