package actions

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/qri-io/dataset"
)

// UnzipGetContents reads a zip file's contents and returns a map from filename to file data
func UnzipGetContents(zipData []byte) (map[string]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}
	// Create a map from filenames in the zip to their json encoded contents.
	contents := make(map[string]string)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, err
		}
		contents[f.Name] = string(data)
	}
	return contents, nil
}

// UnzipDataset reads a zip file from a filename and returns a full dataset with components
func UnzipDataset(zipFilePath string, dsp *dataset.DatasetPod) error {
	zipData, err := ioutil.ReadFile(zipFilePath)
	if err != nil {
		return err
	}

	contents, err := UnzipGetContents(zipData)
	if err != nil {
		return err
	}

	fileData, ok := contents["dataset.json"]
	if !ok {
		return fmt.Errorf("no dataset.json found in the provided zip")
	}
	if err = json.Unmarshal([]byte(fileData), dsp); err != nil {
		return err
	}

	bodyData, ok := contents["body.json"]
	if !ok {
		return fmt.Errorf("no body.json found in the provided zip")
	}
	dsp.BodyBytes = []byte(bodyData)
	dsp.BodyPath = ""

	// TODO: dsp.Transform, dsp.Viz support. Currently no way to set either from bytes, all
	// code assumes they can be referenced by an existing path, but this isn't true for an import.
	dsp.Transform = nil
	dsp.Viz = nil

	// Get ref to existing dataset
	refText, ok := contents["ref.txt"]
	if !ok {
		return fmt.Errorf("no ref.txt found in the provided zip")
	}
	atPos := strings.Index(refText, "@")
	if atPos == -1 {
		return fmt.Errorf("invalid dataset ref: no '@' found")
	}
	// Get name and peername
	datasetName := refText[:atPos]
	sepPos := strings.Index(datasetName, "/")
	if sepPos == -1 {
		return fmt.Errorf("invalid dataset name: no '/' found")
	}
	dsp.Peername = datasetName[:sepPos]
	dsp.Name = datasetName[sepPos+1:]
	return nil
}
