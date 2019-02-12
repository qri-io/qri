package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	testrepo "github.com/qri-io/qri/repo/test"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestExport(t *testing.T) {

	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}

	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	req := NewExportRequests(node, nil)

	var fileWritten string

	tmpDir, err := ioutil.TempDir(os.TempDir(), "export")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		description string
		params      ExportParams
		output      string // error or output filename
	}{
		{"empty ref error", ExportParams{}, "repo: empty dataset reference"},

		{"export default", ExportParams{Ref: "peer/movies"},
			"peer-movies_-_0001-01-01-00-00-00.json"},

		{"export yaml", ExportParams{Ref: "peer/movies", Format: "yaml"},
			"peer-movies_-_0001-01-01-00-00-00.yaml"},

		{"set output name", ExportParams{Ref: "peer/movies", Output: "ds.json"}, "ds.json"},

		{"already exists", ExportParams{Ref: "peer/movies", Output: "ds.json"},
			"already exists: \"ds.json\""},

		{"set output name, yaml", ExportParams{Ref: "peer/movies", Output: "ds.yaml"}, "ds.yaml"},

		{"output to directory", ExportParams{Ref: "peer/cities", Output: "./"},
			"peer-cities_-_0001-01-01-00-00-00.json"},

		{"export xlsx", ExportParams{Ref: "peer/movies", Format: "xlsx"},
			"peer-movies_-_0001-01-01-00-00-00.xlsx"},
	}

	for _, c := range cases {
		c.params.TargetDir = tmpDir
		err := req.Export(&c.params, &fileWritten)

		if err != nil {
			if c.output != err.Error() {
				t.Errorf("case \"%s\" error mismatch: expected: \"%s\", got: \"%s\"",
					c.description, c.output, err)
			}
			continue
		}

		if c.output != fileWritten {
			t.Errorf("case \"%s\" incorrect output: expected: \"%s\", got: \"%s\"",
				c.description, c.output, fileWritten)
		}

		var ds dataset.Dataset
		err = readDataset(filepath.Join(c.params.TargetDir, fileWritten), &ds)
		if err != nil {
			if err.Error() == "SKIP" {
				continue
			}
			t.Fatal(err)
		}

		ref := fmt.Sprintf("%s/%s", ds.Peername, ds.Name)
		if c.params.Ref != ref {
			t.Errorf("case \"%s\" mismatched ds name: expected: \"%s\", got: \"%s\"",
				c.description, c.params.Ref, ref)
		}
	}
}

func readDataset(path string, ds *dataset.Dataset) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	buffer, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		err = json.Unmarshal(buffer, ds)
		if err != nil {
			return err
		}
	case ".yaml":
		err = yaml.Unmarshal(buffer, ds)
		if err != nil {
			return err
		}
	case ".xlsx":
		return fmt.Errorf("SKIP")
	default:
		return fmt.Errorf("unknown format: %s", ext)
	}

	return nil
}
