package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestExport(t *testing.T) {
	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Time{} }
	defer func() { dsfs.Timestamp = prevTs }()

	mr, err := testrepo.NewTestRepo()
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

	if err := ioutil.WriteFile(filepath.Join(tmpDir, "existing_file.json"), []byte("true"), 0664); err != nil {
		t.Fatal(err)
	}

	bad := []struct {
		params ExportParams
		err    string // error or output filename
	}{
		{ExportParams{},
			"repo: empty dataset reference",
		},
		{ExportParams{Ref: "peer/movies", Output: "existing_file.json"},
			`already exists: "existing_file.json"`,
		},
	}

	for _, c := range bad {
		c.params.TargetDir = tmpDir

		t.Run(c.err, func(t *testing.T) {
			err := req.Export(&c.params, &fileWritten)

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if diff := cmp.Diff(c.err, err.Error()); diff != "" {
				t.Errorf("error response mismatch (-want +got):\n%s", diff)
			}
		})
	}

	good := []struct {
		description string
		params      ExportParams
		filename    string
	}{
		{"set output name",
			ExportParams{Ref: "peer/movies", Output: "ds.json"},
			"ds.json",
		},
		{"set output name, yaml",
			ExportParams{Ref: "peer/movies", Output: "ds.yaml"},
			"ds.yaml",
		},
		{"output to directory",
			ExportParams{Ref: "peer/cities", Output: "./"},
			"peer-cities_-_0001-01-01-00-00-00.json",
		},
		{"export xlsx",
			ExportParams{Ref: "peer/movies", Format: "xlsx"},
			"peer-movies_-_0001-01-01-00-00-00.xlsx",
		},
		{"export zip",
			ExportParams{Ref: "peer/movies", Format: "zip"},
			"peer-movies_-_0001-01-01-00-00-00.zip",
		},
		{"export zip",
			ExportParams{Ref: "peer/cities", Zipped: true},
			"peer-cities_-_0001-01-01-00-00-00.zip",
		},
		{"export zip, yaml",
			ExportParams{Ref: "peer/counter", Format: "yaml", Zipped: true},
			"peer-counter_-_0001-01-01-00-00-00.zip",
		},
		{"export zip",
			ExportParams{Ref: "peer/sitemap", Format: "zip", Zipped: true},
			"peer-sitemap_-_0001-01-01-00-00-00.zip",
		},
	}

	for _, c := range good {
		c.params.TargetDir = tmpDir

		t.Run(c.description, func(t *testing.T) {
			err := req.Export(&c.params, &fileWritten)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if c.filename != fileWritten {
				t.Errorf(`incorrect filename. expected: "%s", got: "%s"`, c.filename, fileWritten)
			}

			var ds dataset.Dataset
			err = readDataset(filepath.Join(c.params.TargetDir, fileWritten), &ds)
			if err != nil {
				if err.Error() == "SKIP" {
					return
				}
				t.Fatal(err)
			}

			ref := fmt.Sprintf("%s/%s", ds.Peername, ds.Name)
			if c.params.Ref != ref {
				t.Errorf("case \"%s\" mismatched ds name: expected: \"%s\", got: \"%s\"",
					c.description, c.params.Ref, ref)
			}
		})
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
	case ".zip":
		// TODO: Instead, unzip the file, and inspect the dataset contents.
		return fmt.Errorf("SKIP")
	default:
		return fmt.Errorf("unknown format: %s", ext)
	}

	return nil
}

func TestGenerateFilename(t *testing.T) {
	// no commit
	// no structure & no format
	// no format & yes structure
	// timestamp and format!
	loc := time.FixedZone("UTC-8", -8*60*60)
	timeStamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, loc)
	cases := []struct {
		description string
		ds          *dataset.Dataset
		format      string
		expected    string
		err         string
	}{
		{
			"no format & no structure",
			&dataset.Dataset{}, "", "", "no format specified and no format present in the dataset Structure",
		},
		{
			"no format & no Structure.Format",
			&dataset.Dataset{
				Structure: &dataset.Structure{
					Format: "",
				},
			}, "", "", "no format specified and no format present in the dataset Structure",
		},
		{
			"no format specified & Structure.Format exists",
			&dataset.Dataset{
				Commit: &dataset.Commit{
					Timestamp: timeStamp,
				},
				Structure: &dataset.Structure{
					Format: "json",
				},
				Peername: "cassie",
				Name:     "fun_dataset",
			}, "", "cassie-fun_dataset_-_2009-11-10-23-00-00.json", "",
		},
		{
			"no format specified & Structure.Format exists",
			&dataset.Dataset{
				Commit: &dataset.Commit{
					Timestamp: timeStamp,
				},
				Structure: &dataset.Structure{
					Format: "json",
				},
				Peername: "brandon",
				Name:     "awesome_dataset",
			}, "", "brandon-awesome_dataset_-_2009-11-10-23-00-00.json", "",
		},
		{
			"format: json",
			&dataset.Dataset{
				Commit: &dataset.Commit{
					Timestamp: timeStamp,
				},
				Peername: "ricky",
				Name:     "rad_dataset",
			}, "json", "ricky-rad_dataset_-_2009-11-10-23-00-00.json", "",
		},
		{
			"format: csv",
			&dataset.Dataset{
				Commit: &dataset.Commit{
					Timestamp: timeStamp,
				},
				Peername: "justin",
				Name:     "cool_dataset",
			}, "csv", "justin-cool_dataset_-_2009-11-10-23-00-00.csv", "",
		},
		{
			"no timestamp",
			&dataset.Dataset{
				Peername: "no",
				Name:     "time",
			}, "csv", "no-time_-_0001-01-01-00-00-00.csv", "",
		},
	}
	for _, c := range cases {
		got, err := GenerateFilename(c.ds, c.format)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case '%s' error mismatched: expected: '%s', got: '%s'", c.description, c.err, err)
		}
		if got != c.expected {
			t.Errorf("case '%s' filename mismatched: expected: '%s', got: '%s'", c.description, c.expected, got)
		}
	}
}
