package dsfs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	ipfs_filestore "github.com/qri-io/qfs/cafs/ipfs"
)

// Test Private Key. peerId: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
var testPk = []byte(`CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`)

func init() {
	data, err := base64.StdEncoding.DecodeString(string(testPk))
	if err != nil {
		log.Error(err.Error())
		panic(err)
	}
	testPk = data

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

	privKey, err := crypto.UnmarshalPrivateKey(testPk)
	if err != nil {
		t.Errorf("error unmarshaling private key: %s", err.Error())
		return
	}

	_, err = CreateDataset(ctx, store, nil, nil, nil, false, false, true)
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
			"/map/QmUYex6ravy71tb4kbTmGq4Amr3v61P5crtgBPyBf6edTQ", nil, 6, ""},
		{"all_fields",
			"/map/QmVzXTaBkeibMzEbLX8Na7d5wFr4pmqAVs7iwy3WH1J4qs", nil, 15, ""},
		{"cities_no_commit_title",
			"/map/QmYEcxjtU7oJXeEtp2B5ZaGbw1g5YEyuf6GhL1R3LBo5fh", nil, 17, ""},
		{"craigslist",
			"/map/QmTDcyxadjji59eLEch6KmwEtTVwRzQC9RjdHFRdncZGxV", nil, 21, ""},
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

		path, err := CreateDataset(ctx, store, tc.Input, c.prev, privKey, false, false, true)
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
	_, err = CreateDataset(ctx, store, ds, nil, privKey, false, false, true)
	if err.Error() != expectedErr {
		t.Errorf("case nil body and previous body files, error mismatch: expected '%s', got '%s'", expectedErr, err.Error())
	}

	// Case: no changes in dataset
	expectedErr = "error saving: no changes detected"
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

	_, err = CreateDataset(ctx, store, ds, dsPrev, privKey, false, false, true)
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
	cases := []struct {
		ds, prev *dataset.Dataset
		force    bool
		expected string
		errMsg   string
	}{
		// empty prev
		{&dataset.Dataset{Meta: &dataset.Meta{Title: "new dataset"}}, &dataset.Dataset{}, false, "created dataset", ""},
		// different datasets
		{&dataset.Dataset{Meta: &dataset.Meta{Title: "changes to dataset"}}, &dataset.Dataset{Meta: &dataset.Meta{Title: "new dataset"}}, false, "Meta: 1 change\n\t- modified title", ""},
		// same datasets
		{&dataset.Dataset{Meta: &dataset.Meta{Title: "same dataset"}}, &dataset.Dataset{Meta: &dataset.Meta{Title: "same dataset"}}, false, "", "no changes detected"},
		// same datasets, forced
		{&dataset.Dataset{Meta: &dataset.Meta{Title: "same dataset"}}, &dataset.Dataset{Meta: &dataset.Meta{Title: "same dataset"}}, true, "forced update", ""},
	}

	for i, c := range cases {
		got, err := generateCommitMsg(c.ds, c.prev, c.force)
		if err != nil && c.errMsg != err.Error() {
			t.Errorf("case %d, error mismatch, expect: %s, got: %s", i, c.errMsg, err.Error())
			continue
		}
		if c.expected != got {
			t.Errorf("case %d, message mismatch, expect: %s, got: %s", i, c.expected, got)
		}
	}
}

func TestCleanTitleAndMessage(t *testing.T) {
	ds := &dataset.Dataset{Commit: &dataset.Commit{}}
	cases := []struct {
		title         string
		message       string
		description   string
		expectTitle   string
		expectMessage string
	}{
		// all should be over 70 characters
		// no title, no message, no description
		{"", "", "", "", ""},
		// no title, no message, description
		{"", "", "This is the description we are adding woooo", "This is the description we are adding woooo", ""},
		// no title, no message, long description
		{"", "", "need to make sure this description is over 70 characters long so that we can test to see if the cleaning of the title works", "need to make sure this description is over 70 characters long so ...", "...that we can test to see if the cleaning of the title works"},
		// no title, message, no description
		{"", "This text should move to the title", "", "This text should move to the title", ""},
		// title, no message, no description
		{"Yay, a title", "", "", "Yay, a title", ""},
		// no title, message, description
		{"", "And this text should stay in the message", "This description should move to the title", "This description should move to the title", "And this text should stay in the message"},
		// title, message, no description
		{"We have a title", "And we have a message", "", "We have a title", "And we have a message"},
		// title, no message, description
		{"We have a title", "", "This description will get squashed which I'm not sure what I feel about that", "We have a title", ""},
		// long title, and message
		{"This title is very long and I want to make sure that it works correctly with also having a message. wooo", "This message should still exist", "", "This title is very long and I want to make sure that it works ...", "...correctly with also having a message. wooo\nThis message should still exist"},
	}
	for i, c := range cases {
		ds.Commit.Title = c.title
		ds.Commit.Message = c.message

		cleanTitleAndMessage(&ds.Commit.Title, &ds.Commit.Message, c.description)
		if c.expectTitle != ds.Commit.Title {
			t.Errorf("case %d, title mismatch, expect: %s, got: %s", i, c.expectTitle, ds.Commit.Title)
		}
		if c.expectMessage != ds.Commit.Message {
			t.Errorf("case %d, message mismatch, expect: %s, got: %s", i, c.expectMessage, ds.Commit.Message)
		}

	}
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
