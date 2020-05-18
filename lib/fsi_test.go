package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	cmp "github.com/google/go-cmp/cmp"
	cmpopts "github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestFSIMethodsWrite(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(config.DefaultConfigForTesting(), node)

	// we need some fsi stuff to fully test remove
	methods := NewFSIMethods(inst)
	// create datasets working directory
	datasetsDir, err := ioutil.TempDir("", "QriTestDatasetRequestsRemove")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(datasetsDir)

	// initialize an example no-history dataset
	initp := &InitFSIDatasetParams{
		Name:   "no_history",
		Dir:    datasetsDir,
		Format: "csv",
		Mkdir:  "no_history",
	}
	var noHistoryName string
	if err := methods.InitDataset(initp, &noHistoryName); err != nil {
		t.Fatal(err)
	}

	// link cities dataset with a checkout
	checkoutp := &CheckoutParams{
		Dir: filepath.Join(datasetsDir, "cities"),
		Ref: "me/cities",
	}
	var out string
	if err := methods.Checkout(checkoutp, &out); err != nil {
		t.Fatal(err)
	}

	// link craigslist with a checkout
	checkoutp = &CheckoutParams{
		Dir: filepath.Join(datasetsDir, "craigslist"),
		Ref: "me/craigslist",
	}
	if err := methods.Checkout(checkoutp, &out); err != nil {
		t.Fatal(err)
	}

	badCases := []struct {
		err    string
		params FSIWriteParams
	}{
		{"repo: empty dataset reference", FSIWriteParams{Ref: ""}},
		{"dataset is required", FSIWriteParams{Ref: "abc/ABC"}},
		// TODO (b5) - this is a bug. should return: "'👋' is not a valid dataset reference"
		{"repo: not found", FSIWriteParams{Ref: "👋", Ds: &dataset.Dataset{}}},
		{"repo: not found", FSIWriteParams{Ref: "abc/ABC", Ds: &dataset.Dataset{}}},
		{"dataset is not linked to the filesystem", FSIWriteParams{Ref: "peer/movies", Ds: &dataset.Dataset{}}},
	}

	for _, c := range badCases {
		t.Run(fmt.Sprintf("bad_case_%s", c.err), func(t *testing.T) {
			res := []StatusItem{}
			err := methods.Write(&c.params, &res)

			if err == nil {
				t.Errorf("expected error. got nil")
				return
			} else if c.err != err.Error() {
				t.Errorf("error mismatch: expected: %s, got: %s", c.err, err)
			}
		})
	}

	goodCases := []struct {
		description string
		params      FSIWriteParams
		res         []StatusItem
	}{
		{"update cities structure",
			FSIWriteParams{Ref: "me/cities", Ds: &dataset.Dataset{Structure: &dataset.Structure{Format: "json"}}},
			[]StatusItem{
				{Component: "meta", Type: "unmodified"},
				{Component: "structure", Type: "modified"},
				{Component: "body", Type: "unmodified"},
			},
		},
		// TODO (b5) - doesn't work yet
		// {"overwrite craigslist body",
		// 	FSIWriteParams{Ref: "me/craigslist", Ds: &dataset.Dataset{Body: []interface{}{[]interface{}{"foo", "bar", "baz"}}}},
		// 	[]StatusItem{},
		// },
		{"set title for no history dataset",
			FSIWriteParams{Ref: noHistoryName, Ds: &dataset.Dataset{Meta: &dataset.Meta{Title: "Changed Title"}}},
			[]StatusItem{
				{Component: "meta", Type: "add"},
				{Component: "structure", Type: "add"},
				{Component: "body", Type: "add"},
			},
		},
	}

	for _, c := range goodCases {
		t.Run(fmt.Sprintf("good_case_%s", c.description), func(t *testing.T) {
			res := []StatusItem{}
			err := methods.Write(&c.params, &res)

			if err != nil {
				t.Errorf("unexpected error: %s", err)
				return
			}

			if diff := cmp.Diff(c.res, res, cmpopts.IgnoreFields(StatusItem{}, "Mtime", "SourceFile")); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// Test that checkout requires a valid directory
func TestCheckoutInvalidDirs(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	// Save a dataset with one version.
	run.MustSaveFromBody(t, "movie_ds", "testdata/cities_2/body.csv")

	run.ChdirToRoot()

	// Checkout fails with a blank directory
	err := run.Checkout("me/movie_ds", "")
	if err == nil {
		t.Fatal("expected error from checkout, did not get one")
	}
	expectErr := `need Dir to be a non-empty, absolute path`
	if diff := cmp.Diff(expectErr, err.Error()); diff != "" {
		t.Errorf("error mismatch (-want +got):\n%s", diff)
	}

	// Checkout fails with a relative directory
	err = run.Checkout("me/movie_ds", "relative/dir/")
	if err == nil {
		t.Fatal("expected error from checkout, did not get one")
	}
	expectErr = `need Dir to be a non-empty, absolute path`
	if diff := cmp.Diff(expectErr, err.Error()); diff != "" {
		t.Errorf("error mismatch (-want +got):\n%s", diff)
	}

	// Checkout with an absolute path succeeds
	checkoutPath := filepath.Join(run.TmpDir, "movie_ds")
	err = run.Checkout("me/movie_ds", checkoutPath)
	if err != nil {
		t.Errorf("checkout err: %s", err)
	}
}

// Test that FSI checkout modifies dscache if it exists
func TestDscacheCheckout(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	// Save a dataset with one version.
	_, err := run.SaveWithParams(&SaveParams{
		Ref:        "me/cities_ds",
		BodyPath:   "testdata/cities_2/body.csv",
		UseDscache: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	run.ChdirToRoot()

	// Checkout the dataset, which should update the dscache
	checkoutPath := filepath.Join(run.TmpDir, "cities_ds")
	run.Checkout("me/cities_ds", checkoutPath)

	// Access the dscache
	cache := run.Instance.Dscache()

	// Dscache should have one entry, with an fsiPath set
	actual := run.NiceifyTempDirs(cache.VerboseString(false))
	expect := `Dscache:
 Dscache.Users:
  0) user=peer profileID=QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
 Dscache.Refs:
  0) initID        = vrh4iurbzeyx42trlddzvtoiqevmy2d3mxex4ojd4mxv7cudhlwq
     profileID     = QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = cities_ds
     bodySize      = 155
     bodyRows      = 5
     commitTime    = 978310861
     headRef       = /map/QmPd5jh8ZTFwgbpHaNSu6u2277BKWaiBsqX4uFw7rGWNLu
     fsiPath       = /tmp/cities_ds
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

// Test that FSI init modifies dscache if it exists
func TestDscacheInit(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	// Save a dataset with one version.
	_, err := run.SaveWithParams(&SaveParams{
		Ref:        "me/cities_ds",
		BodyPath:   "testdata/cities_2/body.csv",
		UseDscache: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	workDir := run.CreateAndChdirToWorkDir("json_body")
	_ = workDir

	// Init a new dataset, which should update the dscache
	err = run.Init("me/new_ds", "csv")
	if err != nil {
		t.Fatal(err)
	}

	// Access the dscache
	cache := run.Instance.Dscache()

	// Dscache should have two entries, one has a version, the other has an fsiPath
	actual := run.NiceifyTempDirs(cache.VerboseString(false))
	expect := `Dscache:
 Dscache.Users:
  0) user=peer profileID=QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
 Dscache.Refs:
  0) initID        = vrh4iurbzeyx42trlddzvtoiqevmy2d3mxex4ojd4mxv7cudhlwq
     profileID     = QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = cities_ds
     bodySize      = 155
     bodyRows      = 5
     commitTime    = 978310861
     headRef       = /map/QmPd5jh8ZTFwgbpHaNSu6u2277BKWaiBsqX4uFw7rGWNLu
  1) initID        = ekwzgcu4s4o4xchsoip3oa3j45ko5n7pybtizgvsbudojbhxuita
     profileID     = QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
     topIndex      = 0
     cursorIndex   = 0
     prettyName    = new_ds
     commitTime    = -62135596800
     fsiPath       = /tmp/json_body
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}
