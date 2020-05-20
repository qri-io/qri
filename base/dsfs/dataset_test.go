package dsfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/dataset/generate"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	ipfs_filestore "github.com/qri-io/qfs/cafs/ipfs"
	"github.com/qri-io/qri/base/toqtype"
	testPeers "github.com/qri-io/qri/config/test"
)

func init() {
	// call LoadPlugins once with the empty string b/c we only rely on standard
	// plugins
	if err := ipfs_filestore.LoadPlugins(""); err != nil {
		panic(err)
	}
}

func TestLoadDataset(t *testing.T) {
	ctx := context.Background()
	store := cafs.NewMapstore()
	dsData, err := ioutil.ReadFile("testdata/all_fields/input.dataset.json")
	if err != nil {
		t.Errorf("error loading test dataset: %s", err.Error())
		return
	}
	ds := &dataset.Dataset{}
	if err := ds.UnmarshalJSON(dsData); err != nil {
		t.Errorf("error unmarshaling test dataset: %s", err.Error())
		return
	}
	body, err := ioutil.ReadFile("testdata/all_fields/body.csv")
	if err != nil {
		t.Errorf("error loading test body: %s", err.Error())
		return
	}

	ds.SetBodyFile(qfs.NewMemfileBytes("all_fields.csv", body))

	apath, err := WriteDataset(ctx, store, ds, true)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	loadedDataset, err := LoadDataset(ctx, store, apath)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	// prove we aren't returning a path to a dataset that ends with `/dataset.json`
	if strings.Contains(loadedDataset.Path, "/dataset.json") {
		t.Errorf("path should not contain the basename of the dataset file: %s", loadedDataset.Path)
	}

	cases := []struct {
		ds  *dataset.Dataset
		err string
	}{
		{dataset.NewDatasetRef("/bad/path"),
			"error loading dataset: error getting file bytes: cafs: path not found"},
		{&dataset.Dataset{
			Meta: dataset.NewMetaRef("/bad/path"),
		}, "error loading dataset metadata: error loading metadata file: cafs: path not found"},
		{&dataset.Dataset{
			Structure: dataset.NewStructureRef("/bad/path"),
		}, "error loading dataset structure: error loading structure file: cafs: path not found"},
		{&dataset.Dataset{
			Structure: dataset.NewStructureRef("/bad/path"),
		}, "error loading dataset structure: error loading structure file: cafs: path not found"},
		{&dataset.Dataset{
			Transform: dataset.NewTransformRef("/bad/path"),
		}, "error loading dataset transform: error loading transform raw data: cafs: path not found"},
		{&dataset.Dataset{
			Commit: dataset.NewCommitRef("/bad/path"),
		}, "error loading dataset commit: error loading commit file: cafs: path not found"},
		{&dataset.Dataset{
			Viz: dataset.NewVizRef("/bad/path"),
		}, "error loading dataset viz: error loading viz file: cafs: path not found"},
	}

	for i, c := range cases {
		path := c.ds.Path
		if !c.ds.IsEmpty() {
			dsf, err := JSONFile(PackageFileDataset.String(), c.ds)
			if err != nil {
				t.Errorf("case %d error generating json file: %s", i, err.Error())
				continue
			}
			path, err = store.Put(ctx, dsf)
			if err != nil {
				t.Errorf("case %d error putting file in store: %s", i, err.Error())
				continue
			}
		}

		_, err = LoadDataset(ctx, store, path)
		if !(err != nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}
	}

}

func TestCreateDataset(t *testing.T) {
	ctx := context.Background()
	store := cafs.NewMapstore()
	prev := Timestamp
	// shameless call to timestamp to get the coverge points
	Timestamp()

	defer func() { Timestamp = prev }()
	Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	// These tests are using hard-coded ids that require this exact peer's private key.
	info := testPeers.GetTestPeerInfo(10)
	privKey := info.PrivKey

	_, err := CreateDataset(ctx, store, nil, nil, nil, SaveSwitches{ShouldRender: true})
	if err == nil {
		t.Errorf("expected call without prvate key to error")
		return
	}
	pkReqErrMsg := "private key is required to create a dataset"
	if err.Error() != pkReqErrMsg {
		t.Errorf("error mismatch. '%s' != '%s'", pkReqErrMsg, err.Error())
		return
	}

	cases := []struct {
		casePath   string
		resultPath string
		prev       *dataset.Dataset
		repoFiles  int // expected total count of files in repo after test execution
		err        string
	}{
		{"invalid_reference",
			"", nil, 0, "error loading dataset commit: error loading commit file: cafs: path not found"},
		{"invalid",
			"", nil, 0, "commit is required"},
		{"strict_fail",
			"", nil, 0, "strict mode: dataset body did not validate against its schema"},
		{"cities",
			"/map/QmXgaGiPcpiRcCkHrt4boC13hrqYshMFe3BztfLXgvh3pF", nil, 6, ""},
		{"all_fields",
			"/map/QmXQeeGnKXvn68uRZYffodggE298BWYFU7st7TPT9j67PL", nil, 15, ""},
		{"cities_no_commit_title",
			"/map/QmSCoZYVStTyUcDZPudPjRtxCPf2xAQNTFmvrLgkufUzJg", nil, 17, ""},
		{"craigslist",
			"/map/QmQsYot555Mn1K6ktrPa6bT3hXMTrEc6Wrp7FpbLb3jQDv", nil, 21, ""},
		// should error when previous dataset won't dereference.
		{"craigslist",
			"", &dataset.Dataset{Structure: dataset.NewStructureRef("/bad/path")}, 21, "error loading dataset structure: error loading structure file: cafs: path not found"},
		// should error when previous dataset isn't valid. Aka, when it isn't empty, but missing
		// either structure or commit. Commit is checked for first.
		{"craigslist",
			"", &dataset.Dataset{Meta: &dataset.Meta{Title: "previous"}, Structure: nil}, 21, "commit is required"},
	}

	for _, c := range cases {
		tc, err := dstest.NewTestCaseFromDir("testdata/" + c.casePath)
		if err != nil {
			t.Errorf("%s: error creating test case: %s", c.casePath, err)
			continue
		}

		path, err := CreateDataset(ctx, store, tc.Input, c.prev, privKey, SaveSwitches{ShouldRender: true})
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("%s: error mismatch. expected: '%s', got: '%s'", tc.Name, c.err, err)
			continue
		} else if c.err != "" {
			continue
		}

		ds, err := LoadDataset(ctx, store, path)
		if err != nil {
			t.Errorf("%s: error loading dataset: %s", tc.Name, err.Error())
			continue
		}
		ds.Path = ""

		if tc.Expect != nil {
			if err := dataset.CompareDatasets(tc.Expect, ds); err != nil {
				// expb, _ := json.Marshal(tc.Expect)
				// fmt.Println(string(expb))
				// dsb, _ := json.Marshal(ds)
				// fmt.Println(string(dsb))
				t.Errorf("%s: dataset comparison error: %s", tc.Name, err.Error())
			}
		}

		if c.resultPath != path {
			t.Errorf("%s: result path mismatch: expected: '%s', got: '%s'", tc.Name, c.resultPath, path)
		}
		if len(store.Files) != c.repoFiles {
			t.Errorf("%s: invalid number of mapstore entries: %d != %d", tc.Name, c.repoFiles, len(store.Files))
			_, err := store.Print()
			if err != nil {
				panic(err)
			}
			continue
		}
	}

	// Case: no body or previous body files
	dsData, err := ioutil.ReadFile("testdata/cities/input.dataset.json")
	if err != nil {
		t.Errorf("case nil body and previous body files, error reading dataset file: %s", err.Error())
	}
	ds := &dataset.Dataset{}
	if err := ds.UnmarshalJSON(dsData); err != nil {
		t.Errorf("case nil body and previous body files, error unmarshaling dataset file: %s", err.Error())
	}

	if err != nil {
		t.Errorf("case nil body and previous body files, error reading data file: %s", err.Error())
	}
	expectedErr := "bodyfile or previous bodyfile needed"
	_, err = CreateDataset(ctx, store, ds, nil, privKey, SaveSwitches{ShouldRender: true})
	if err.Error() != expectedErr {
		t.Errorf("case nil body and previous body files, error mismatch: expected '%s', got '%s'", expectedErr, err.Error())
	}

	// Case: no changes in dataset
	expectedErr = "error saving: no changes"
	dsPrev, err := LoadDataset(ctx, store, cases[3].resultPath)
	ds.PreviousPath = cases[3].resultPath
	if err != nil {
		t.Errorf("case no changes in dataset, error loading previous dataset file: %s", err.Error())
	}

	bodyBytes, err := ioutil.ReadFile("testdata/cities/body.csv")
	if err != nil {
		t.Errorf("case no changes in dataset, error reading body file: %s", err.Error())
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.csv", bodyBytes))

	_, err = CreateDataset(ctx, store, ds, dsPrev, privKey, SaveSwitches{ShouldRender: true})
	if err != nil && err.Error() != expectedErr {
		t.Errorf("case no changes in dataset, error mismatch: expected '%s', got '%s'", expectedErr, err.Error())
	} else if err == nil {
		t.Errorf("case no changes in dataset, expected error got 'nil'")
	}

	if len(store.Files) != 21 {
		t.Errorf("case nil datafile and PreviousPath, invalid number of entries: %d != %d", 20, len(store.Files))
		_, err := store.Print()
		if err != nil {
			panic(err)
		}
	}

	// case: previous dataset isn't valid
}

// Test that if the body is too large, the commit message just assumes the body changed
func TestCreateDatasetBodyTooLarge(t *testing.T) {
	ctx := context.Background()
	store := cafs.NewMapstore()

	prevTs := Timestamp
	defer func() { Timestamp = prevTs }()
	Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	// Set the limit for the body to be 100 bytes
	prevBodySizeLimit := BodySizeSmallEnoughToDiff
	defer func() { BodySizeSmallEnoughToDiff = prevBodySizeLimit }()
	BodySizeSmallEnoughToDiff = 100

	info := testPeers.GetTestPeerInfo(10)
	privKey := info.PrivKey

	// Need a previous commit, otherwise we just get the "created dataset" message
	prevDs := dataset.Dataset{
		Commit: &dataset.Commit{},
		Structure: &dataset.Structure{
			Format: "csv",
			Schema: BaseTabularSchema,
		},
	}

	testBodyPath, _ := filepath.Abs("testdata/movies/body.csv")
	testBodyBytes, _ := ioutil.ReadFile(testBodyPath)

	// Create a new version and add the body
	nextDs := dataset.Dataset{
		Commit: &dataset.Commit{},
		Structure: &dataset.Structure{
			Format: "csv",
			Schema: BaseTabularSchema,
		},
	}
	nextDs.SetBodyFile(qfs.NewMemfileBytes(testBodyPath, testBodyBytes))

	path, err := CreateDataset(ctx, store, &nextDs, &prevDs, privKey, SaveSwitches{ShouldRender: true})
	if err != nil {
		t.Fatalf("CreateDataset: %s", err)
	}

	// Load the created dataset to inspect the commit message
	result, err := LoadDataset(ctx, store, path)
	if err != nil {
		t.Fatalf("LoadDataset: %s", err)
	}

	commitBytes, err := json.Marshal(result.Commit)
	if err != nil {
		t.Fatalf("commit.Marshal: %s", err)
	}
	expectBytes := `{"message":"body changed","path":"/map/Qmf23yGuYSW7uWtDeV8qpJkbch5PQjcJ2GQK1y1C4Ynv4i","qri":"cm:0","signature":"tH8RcOEjyev68eY4SewmDDUnwVqqJX5ckoUzHVGHGn93YW5TQz51EoslLP6ovrOUKx+MaOGqE05bDpD943SDo10eU023R+wEci+V2Z/WFffN0u39tz1a3KObql5kkC+DynXx7UJo/xpfa7KzH/BLvZDtEnwavu3HqL/CoiPEN6bipjlNG7qu47tifAaOQGgCQ2/3Cufm2ejJ90RPUi+kV1/jdPYU3vPu7ijfIAULM+7ZqZ44b4t+B15UUnCaP/Tk8I+/v855v4syZ2iW3UTR0hSyHwdSW2PkxyuRAU0hLSgHS0iw8vwudnjQuFPTd3Cyh+VWf5FDSd7YbO3Oj24JXg==","timestamp":"2001-01-01T01:01:01.000000001Z","title":"body changed"}`

	if diff := cmp.Diff(expectBytes, string(commitBytes)); diff != "" {
		t.Fatalf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestWriteDataset(t *testing.T) {
	ctx := context.Background()
	store := cafs.NewMapstore()
	prev := Timestamp
	defer func() { Timestamp = prev }()
	Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	if _, err := WriteDataset(ctx, store, nil, true); err == nil || err.Error() != "cannot save empty dataset" {
		t.Errorf("didn't reject empty dataset: %s", err)
	}
	if _, err := WriteDataset(ctx, store, &dataset.Dataset{}, true); err == nil || err.Error() != "cannot save empty dataset" {
		t.Errorf("didn't reject empty dataset: %s", err)
	}

	cases := []struct {
		casePath  string
		repoFiles int // expected total count of files in repo after test execution
		err       string
	}{
		{"cities", 6, ""},      // dataset, commit, structure, meta, viz, body
		{"all_fields", 14, ""}, // dataset, commit, structure, meta, viz, viz_script, transform, transform_script, SAME BODY as cities -> gets de-duped
	}

	for i, c := range cases {
		tc, err := dstest.NewTestCaseFromDir("testdata/" + c.casePath)
		if err != nil {
			t.Errorf("%s: error creating test case: %s", c.casePath, err)
			continue
		}

		ds := tc.Input

		got, err := WriteDataset(ctx, store, ds, true)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		// total count expected of files in repo after test execution
		if len(store.Files) != c.repoFiles {
			t.Errorf("case expected %d invalid number of entries: %d != %d", i, c.repoFiles, len(store.Files))
			str, err := store.Print()
			if err != nil {
				panic(err)
			}
			t.Log(str)
			continue
		}

		f, err := store.Get(ctx, got)
		if err != nil {
			t.Errorf("error getting dataset file: %s", err.Error())
			continue
		}

		ref := &dataset.Dataset{}
		if err := json.NewDecoder(f).Decode(ref); err != nil {
			t.Errorf("error decoding dataset json: %s", err.Error())
			continue
		}

		if ref.Transform != nil {
			if !ref.Transform.IsEmpty() {
				t.Errorf("expected stored dataset.Transform to be a reference")
			}
			ds.Transform.Assign(dataset.NewTransformRef(ref.Transform.Path))
		}
		if ref.Meta != nil {
			if !ref.Meta.IsEmpty() {
				t.Errorf("expected stored dataset.Meta to be a reference")
			}
			// Abstract transforms aren't loaded
			ds.Meta.Assign(dataset.NewMetaRef(ref.Meta.Path))
		}
		if ref.Structure != nil {
			if !ref.Structure.IsEmpty() {
				t.Errorf("expected stored dataset.Structure to be a reference")
			}
			ds.Structure.Assign(dataset.NewStructureRef(ref.Structure.Path))
		}
		if ref.Viz != nil {
			if !ref.Viz.IsEmpty() {
				t.Errorf("expected stored dataset.Viz to be a reference")
			}
			ds.Viz.Assign(dataset.NewVizRef(ref.Viz.Path))
		}
		ds.BodyPath = ref.BodyPath

		ds.Assign(dataset.NewDatasetRef(got))
		result, err := LoadDataset(ctx, store, got)
		if err != nil {
			t.Errorf("case %d unexpected error loading dataset: %s", i, err)
			continue
		}

		if err := dataset.CompareDatasets(ds, result); err != nil {
			t.Errorf("case %d comparison mismatch: %s", i, err.Error())

			d1, _ := ds.MarshalJSON()
			t.Log(string(d1))

			d, _ := result.MarshalJSON()
			t.Log(string(d))
			continue
		}
	}
}

func TestGenerateCommitMessage(t *testing.T) {
	badCases := []struct {
		description string
		prev, ds    *dataset.Dataset
		force       bool
		errMsg      string
	}{
		{
			"no changes from one dataset version to next",
			&dataset.Dataset{Meta: &dataset.Meta{Title: "same dataset"}},
			&dataset.Dataset{Meta: &dataset.Meta{Title: "same dataset"}},
			false,
			"no changes",
		},
	}

	store := cafs.NewMapstore()

	for _, c := range badCases {
		t.Run(fmt.Sprintf("%s", c.description), func(t *testing.T) {
			_, _, err := generateCommitDescriptions(store, c.prev, c.ds, BodySame, c.force)
			if err == nil {
				t.Errorf("error expected, did not get one")
			} else if c.errMsg != err.Error() {
				t.Errorf("error mismatch\nexpect: %s\ngot: %s", c.errMsg, err.Error())
			}
		})
	}

	goodCases := []struct {
		description string
		prev, ds    *dataset.Dataset
		force       bool
		expectShort string
		expectLong  string
	}{
		{
			"empty previous and non-empty dataset",
			&dataset.Dataset{},
			&dataset.Dataset{Meta: &dataset.Meta{Title: "new dataset"}},
			false,
			"created dataset",
			"created dataset",
		},
		{
			"title changes from previous",
			&dataset.Dataset{Meta: &dataset.Meta{Title: "new dataset"}},
			&dataset.Dataset{Meta: &dataset.Meta{Title: "changes to dataset"}},
			false,
			"meta updated title",
			"meta:\n\tupdated title",
		},
		{
			"same dataset but force is true",
			&dataset.Dataset{Meta: &dataset.Meta{Title: "same dataset"}},
			&dataset.Dataset{Meta: &dataset.Meta{Title: "same dataset"}},
			true,
			"forced update",
			"forced update",
		},
		{
			"structure sets the headerRow config option",
			&dataset.Dataset{Structure: &dataset.Structure{
				FormatConfig: map[string]interface{}{
					"headerRow": false,
				},
			}},
			&dataset.Dataset{Structure: &dataset.Structure{
				FormatConfig: map[string]interface{}{
					"headerRow": true,
				},
			}},
			false,
			"structure updated formatConfig.headerRow",
			"structure:\n\tupdated formatConfig.headerRow",
		},
		{
			"readme modified",
			&dataset.Dataset{Readme: &dataset.Readme{
				Format:      "md",
				ScriptBytes: []byte("# hello\n\ncontent\n\n"),
			}},
			&dataset.Dataset{Readme: &dataset.Readme{
				Format:      "md",
				ScriptBytes: []byte("# hello\n\ncontent\n\nanother line\n\n"),
			}},
			false,
			// TODO(dlong): Should mention the line added.
			"readme updated scriptBytes",
			"readme:\n\tupdated scriptBytes",
		},
		{
			"body with a small number of changes",
			&dataset.Dataset{
				Structure: &dataset.Structure{Format: "json"},
				Body: toqtype.MustParseJSONAsArray(`[
  { "fruit": "apple", "color": "red" },
  { "fruit": "banana", "color": "yellow" },
  { "fruit": "cherry", "color": "red" }
]`),
			},
			&dataset.Dataset{
				Structure: &dataset.Structure{Format: "json"},
				Body: toqtype.MustParseJSONAsArray(`[
  { "fruit": "apple", "color": "red" },
  { "fruit": "blueberry", "color": "blue" },
  { "fruit": "cherry", "color": "red" },
  { "fruit": "durian", "color": "green" }
]`),
			},
			false,
			"body updated row 1 and added row 3",
			"body:\n\tupdated row 1\n\tadded row 3",
		},
		{
			"body with lots of changes",
			&dataset.Dataset{
				Structure: &dataset.Structure{Format: "csv"},
				Body: toqtype.MustParseCsvAsArray(`one,two,3
four,five,6
seven,eight,9
ten,eleven,12
thirteen,fourteen,15
sixteen,seventeen,18
nineteen,twenty,21
twenty-two,twenty-three,24
twenty-five,twenty-six,27
twenty-eight,twenty-nine,30`),
			},
			&dataset.Dataset{
				Structure: &dataset.Structure{Format: "csv"},
				Body: toqtype.MustParseCsvAsArray(`one,two,3
four,five,6
seven,eight,cat
dog,eleven,12
thirteen,eel,15
sixteen,seventeen,100
frog,twenty,21
twenty-two,twenty-three,24
twenty-five,giraffe,200
hen,twenty-nine,30`),
			},
			false,
			"body changed by 19%",
			"body:\n\tchanged by 19%",
		},
		{
			"meta and structure and readme changes",
			&dataset.Dataset{
				Meta: &dataset.Meta{Title: "new dataset"},
				Structure: &dataset.Structure{
					FormatConfig: map[string]interface{}{
						"headerRow": false,
					},
				},
				Readme: &dataset.Readme{
					Format:      "md",
					ScriptBytes: []byte("# hello\n\ncontent\n\n"),
				},
			},
			&dataset.Dataset{
				Meta: &dataset.Meta{Title: "changes to dataset"},
				Structure: &dataset.Structure{
					FormatConfig: map[string]interface{}{
						"headerRow": true,
					},
				},
				Readme: &dataset.Readme{
					Format:      "md",
					ScriptBytes: []byte("# hello\n\ncontent\n\nanother line\n\n"),
				},
			},
			false,
			"updated meta, structure, and readme",
			"meta:\n\tupdated title\nstructure:\n\tupdated formatConfig.headerRow\nreadme:\n\tupdated scriptBytes",
		},
		{
			"meta removed but everything else is the same",
			&dataset.Dataset{
				Meta: &dataset.Meta{Title: "new dataset"},
				Structure: &dataset.Structure{
					FormatConfig: map[string]interface{}{
						"headerRow": false,
					},
				},
				Readme: &dataset.Readme{
					Format:      "md",
					ScriptBytes: []byte("# hello\n\ncontent\n\n"),
				},
			},
			&dataset.Dataset{
				Structure: &dataset.Structure{
					FormatConfig: map[string]interface{}{
						"headerRow": false,
					},
				},
				Readme: &dataset.Readme{
					Format:      "md",
					ScriptBytes: []byte("# hello\n\ncontent\n\n"),
				},
			},
			false,
			"meta removed",
			"meta removed",
		},
		{
			"meta has multiple parts changed",
			&dataset.Dataset{
				Meta: &dataset.Meta{
					Title:       "new dataset",
					Description: "TODO: Add description",
				},
			},
			&dataset.Dataset{
				Meta: &dataset.Meta{
					Title:       "changes to dataset",
					HomeURL:     "http://example.com",
					Description: "this is a great description",
				},
			},
			false,
			"meta updated 3 fields",
			"meta:\n\tupdated description\n\tadded homeURL\n\tupdated title",
		},
		{
			"meta and body changed",
			&dataset.Dataset{
				Meta: &dataset.Meta{
					Title:       "new dataset",
					Description: "TODO: Add description",
				},
				Structure: &dataset.Structure{Format: "csv"},
				Body: toqtype.MustParseCsvAsArray(`one,two,3
four,five,6
seven,eight,9
ten,eleven,12
thirteen,fourteen,15
sixteen,seventeen,18
nineteen,twenty,21
twenty-two,twenty-three,24
twenty-five,twenty-six,27
twenty-eight,twenty-nine,30`),
			},
			&dataset.Dataset{
				Meta: &dataset.Meta{
					Title:       "changes to dataset",
					HomeURL:     "http://example.com",
					Description: "this is a great description",
				},
				Structure: &dataset.Structure{Format: "csv"},
				Body: toqtype.MustParseCsvAsArray(`one,two,3
four,five,6
something,eight,cat
dog,eleven,12
thirteen,eel,15
sixteen,60,100
frog,twenty,21
twenty-two,twenty-three,24
twenty-five,giraffe,200
hen,twenty-nine,30`),
			},
			false,
			"updated meta and body",
			"meta:\n\tupdated description\n\tadded homeURL\n\tupdated title\nbody:\n\tchanged by 24%",
		},
		{
			"meta changed but body stays the same",
			&dataset.Dataset{
				Meta: &dataset.Meta{
					Title: "new dataset",
				},
				Structure: &dataset.Structure{Format: "csv"},
				Body: toqtype.MustParseCsvAsArray(`one,two,3
four,five,6
seven,eight,9
ten,eleven,12
thirteen,fourteen,15
sixteen,seventeen,18`),
			},
			&dataset.Dataset{
				Meta: &dataset.Meta{
					Title: "dataset of a bunch of numbers",
				},
				Structure: &dataset.Structure{Format: "csv"},
				Body: toqtype.MustParseCsvAsArray(`one,two,3
four,five,6
seven,eight,9
ten,eleven,12
thirteen,fourteen,15
sixteen,seventeen,18`),
			},
			false,
			"meta updated title",
			"meta:\n\tupdated title",
		},
	}

	for _, c := range goodCases {
		t.Run(c.description, func(t *testing.T) {
			bodyAct := BodyDefault
			if compareBody(c.prev.Body, c.ds.Body) {
				bodyAct = BodySame
			}
			shortTitle, longMessage, err := generateCommitDescriptions(store, c.prev, c.ds, bodyAct, c.force)
			if err != nil {
				t.Errorf("error: %s", err.Error())
				return
			}
			if c.expectShort != shortTitle {
				t.Errorf("short message mismatch\nexpect: %s\ngot: %s", c.expectShort, shortTitle)
			}
			if c.expectLong != longMessage {
				t.Errorf("long message mismatch\nexpect: %s\ngot: %s", c.expectLong, longMessage)
			}
		})
	}
}

func compareBody(left, right interface{}) bool {
	leftData, err := json.Marshal(left)
	if err != nil {
		panic(err)
	}
	rightData, err := json.Marshal(right)
	if err != nil {
		panic(err)
	}
	return string(leftData) == string(rightData)
}

func TestGetDepth(t *testing.T) {
	good := []struct {
		val      string
		expected int
	}{
		{`"foo"`, 0},
		{`1000`, 0},
		{`true`, 0},
		{`{"foo": "bar"}`, 1},
		{`{"foo": "bar","bar": "baz"}`, 1},
		{`{
			"foo":"bar",
			"bar": "baz",
			"baz": {
				"foo": "bar",
				"bar": "baz"
			}
		}`, 2},
		{`{
			"foo": "bar",
			"bar": "baz",
			"baz": {
				"foo": "bar",
				"bar": [
					"foo",
					"bar",
					"baz"
				]
			}
		}`, 3},
		{`{
			"foo": "bar",
			"bar": "baz",
			"baz": [
				"foo",
				"bar",
				"baz"
			]
		}`, 2},
		{`["foo","bar","baz"]`, 1},
		{`["a","b",[1, 2, 3]]`, 2},
		{`[
			"foo",
			"bar",
			{"baz": {
				"foo": "bar",
				"bar": "baz",
				"baz": "foo"
				}
			}
		]`, 3},
		{`{
			"foo": "bar",
			"foo1": {
				"foo2": 2,
				"foo3": false
			},
			"foo4": "bar",
			"foo5": {
				"foo6": 100
			}
		}`, 2},
		{`{
			"foo":  "bar",
			"foo1": "bar",
			"foo2": {
				"foo3": 100,
				"foo4": 100
			},
			"foo5": {
				"foo6": 100,
				"foo7": 100,
				"foo8": 100,
				"foo9": 100
			},
			"foo10": {
				"foo11": 100,
				"foo12": 100
			}
		}`, 2},
	}

	var val interface{}

	for i, c := range good {
		if err := json.Unmarshal([]byte(c.val), &val); err != nil {
			t.Fatal(err)
		}
		depth := getDepth(val)
		if c.expected != depth {
			t.Errorf("case %d, depth mismatch, expected %d, got %d", i, c.expected, depth)
		}
	}
}

func GenerateDataset(b *testing.B, sampleSize int, format string) (int, *dataset.Dataset) {
	ds := &dataset.Dataset{
		Commit: &dataset.Commit{
			Timestamp: time.Date(2017, 1, 1, 1, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Meta: &dataset.Meta{
			Title: "performance benchmark data",
		},
		Structure: &dataset.Structure{
			Format: format,
			FormatConfig: map[string]interface{}{
				"headerRow":  true,
				"lazyQuotes": true,
			},
			Schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "array",
					"items": []interface{}{
						map[string]interface{}{"title": "uuid", "type": "string"},
						map[string]interface{}{"title": "ingest", "type": "string"},
						map[string]interface{}{"title": "occurred", "type": "string"},
						map[string]interface{}{"title": "raw_data", "type": "string"},
					},
				},
			},
		},
	}

	gen, err := generate.NewTabularGenerator(ds.Structure)
	if err != nil {
		b.Errorf("error creating generator: %s", err.Error())
	}
	defer gen.Close()

	bodyBuffer := &bytes.Buffer{}
	w, err := dsio.NewEntryWriter(ds.Structure, bodyBuffer)
	if err != nil {
		b.Fatalf("creating entry writer: %s", err.Error())
	}

	for i := 0; i < sampleSize; i++ {
		ent, err := gen.ReadEntry()
		if err != nil {
			b.Fatalf("reading generator entry: %s", err.Error())
		}
		w.WriteEntry(ent)
	}
	if err := w.Close(); err != nil {
		b.Fatalf("closing writer: %s", err)
	}

	fileName := fmt.Sprintf("body.%s", ds.Structure.Format)
	ds.SetBodyFile(qfs.NewMemfileReader(fileName, bodyBuffer))

	return bodyBuffer.Len(), ds
}

func BenchmarkCreateDatasetCSV(b *testing.B) {
	// ~1 MB, ~12 MB, ~25 MB, ~50 MB, ~500 MB, ~1GB
	for _, sampleSize := range []int{10000, 100000, 250000, 500000, 1000000} {
		ctx := context.Background()
		store := cafs.NewMapstore()
		prev := Timestamp

		defer func() { Timestamp = prev }()
		Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

		// These tests are using hard-coded ids that require this exact peer's private key.
		info := testPeers.GetTestPeerInfo(10)
		privKey := info.PrivKey

		b.Run(fmt.Sprintf("sample size %v", sampleSize), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()

				_, dataset := GenerateDataset(b, sampleSize, "csv")

				b.StartTimer()
				_, err := CreateDataset(ctx, store, dataset, nil, privKey, SaveSwitches{ShouldRender: true})
				if err != nil {
					b.Errorf("error creating dataset: %s", err.Error())
				}
			}
			b.StopTimer()
		})
	}
}

// validateDataset is a stripped copy of base/dsfs/setErrCount
func validateDataset(ds *dataset.Dataset, data qfs.File) error {
	defer data.Close()

	er, err := dsio.NewEntryReader(ds.Structure, data)
	if err != nil {
		return err
	}

	_, err = validate.EntryReader(er)

	return err
}

func BenchmarkValidateCSV(b *testing.B) {
	// ~1 MB, ~12 MB, ~25 MB, ~50 MB, ~500 MB, ~1GB
	for _, sampleSize := range []int{10000, 100000, 250000, 500000, 1000000, 10000000} {
		b.Run(fmt.Sprintf("sample size %v", sampleSize), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				_, dataset := GenerateDataset(b, sampleSize, "csv")

				b.StartTimer()
				err := validateDataset(dataset, dataset.BodyFile())
				if err != nil {
					b.Errorf("error creating dataset: %s", err.Error())
				}
			}
			b.StopTimer()
		})
	}
}

func BenchmarkValidateJSON(b *testing.B) {
	// ~1 MB, ~12 MB, ~25 MB, ~50 MB, ~500 MB, ~1GB
	for _, sampleSize := range []int{10000, 100000, 250000, 500000, 1000000, 10000000} {
		b.Run(fmt.Sprintf("sample size %v", sampleSize), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				_, dataset := GenerateDataset(b, sampleSize, "json")

				b.StartTimer()
				err := validateDataset(dataset, dataset.BodyFile())
				if err != nil {
					b.Errorf("error creating dataset: %s", err.Error())
				}
			}
			b.StopTimer()
		})
	}
}
