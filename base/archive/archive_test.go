package archive

import (
	"context"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/dsfs"
	testPeers "github.com/qri-io/qri/config/test"
)

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

func testFS() (qfs.Filesystem, map[string]string, error) {
	ctx := context.Background()
	dataf := qfs.NewMemfileBytes("movies.csv", []byte("movie\nup\nthe incredibles"))

	// Map strings to ds.keys for convenience
	ns := map[string]string{
		"movies": "",
	}

	ds := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "csv",
			Schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "array",
					"items": []interface{}{
						map[string]interface{}{"title": "movie", "type": "string"},
					},
				},
			},
		},
	}
	ds.SetBodyFile(dataf)

	fs := qfs.NewMemFS()
	dskey, err := dsfs.WriteDataset(ctx, fs, ds, true)
	if err != nil {
		return fs, ns, err
	}
	ns["movies"] = dskey

	return fs, ns, nil
}

func testFSWithVizAndTransform() (qfs.Filesystem, map[string]string, error) {
	ctx := context.Background()
	ds := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "csv",
			Schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "array",
					"items": []interface{}{
						map[string]interface{}{"title": "movie", "type": "string"},
					},
				},
			},
		},
		Transform: &dataset.Transform{
			ScriptPath:  "transform_script",
			ScriptBytes: []byte("def transform(ds):\nreturn ds\n"),
		},
		Viz: &dataset.Viz{
			ScriptPath:  "viz_script",
			ScriptBytes: []byte("<html>template</html>\n"),
		},
	}
	// load scripts into file pointers, time for a NewDataset function?
	ds.Transform.OpenScriptFile(ctx, nil)
	ds.Viz.OpenScriptFile(ctx, nil)
	ds.Viz.SetRenderedFile(qfs.NewMemfileBytes("index.html", []byte("<html>rendered</html<\n")))

	// Map strings to ds.keys for convenience
	ns := map[string]string{}
	// Store the files
	st := qfs.NewMemFS()
	ds.SetBodyFile(qfs.NewMemfileBytes("movies.csv", []byte("movie\nup\nthe incredibles")))
	privKey := testPeers.GetTestPeerInfo(10).PrivKey

	var dsLk sync.Mutex
	dskey, err := dsfs.WriteDataset(ctx, dsLk, fs, ds, privKey, dsfs.SaveSwitches{Pin: true})
	if err != nil {
		return st, ns, err
	}
	ns["movies"] = dskey
	ns["transform_script"] = ds.Transform.ScriptPath
	ns["viz_template"] = ds.Viz.ScriptPath
	ns["index.html"] = ds.Viz.RenderedPath
	return st, ns, nil
}

func zipTestdataFile(path string) string {
	_, currfile, _, _ := runtime.Caller(0)
	testdataPath := filepath.Join(filepath.Dir(currfile), "../dsfs/testdata/zip")
	return filepath.Join(testdataPath, path)
}
