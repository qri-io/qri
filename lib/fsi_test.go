package lib

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	cmp "github.com/google/go-cmp/cmp"
	cmpopts "github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/p2p"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestFSIMethodsWrite(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)

	// we need some fsi stuff to fully test remove
	methods := NewFSIMethods(inst)
	// create datasets working directory
	datasetsDir, err := ioutil.TempDir("", "QriTestDatasetRequestsRemove")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(datasetsDir)

	// initialize an example no-history dataset
	initp := &InitDatasetParams{
		Name:      "no_history",
		TargetDir: filepath.Join(datasetsDir, "no_history"),
		Format:    "csv",
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
		{"empty reference", FSIWriteParams{Ref: ""}},
		{"dataset name may not contain any upper-case letters", FSIWriteParams{Ref: "abc/ABC"}},
		{"dataset is required", FSIWriteParams{Ref: "abc/movies"}},
		{"unexpected character at position 0: '\xc3\xb0'", FSIWriteParams{Ref: "👋", Ds: &dataset.Dataset{}}},
		{"repo: not found", FSIWriteParams{Ref: "abc/movies", Ds: &dataset.Dataset{}}},
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
	checkoutPath := PathJoinPosix(run.TmpDir, "cities_ds")
	run.Checkout("me/cities_ds", checkoutPath)

	// Access the dscache
	cache := run.Instance.Dscache()

	// Dscache should have one entry, with an fsiPath set
	actual := run.NiceifyTempDirs(cache.VerboseString(false))
	expect := dstest.Template(t, `Dscache:
 Dscache.Users:
  0) user=peer profileID={{ .profileID }}
 Dscache.Refs:
  0) initID        = vrh4iurbzeyx42trlddzvtoiqevmy2d3mxex4ojd4mxv7cudhlwq
     profileID     = {{ .profileID }}
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = cities_ds
     bodySize      = 155
     bodyRows      = 5
     commitTime    = 978310861
     headRef       = {{ .headRef }}
     fsiPath       = /tmp/cities_ds
`, map[string]string{
		"profileID": "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
		"headRef":   "/mem/QmQqnpYhA7UfcyCLkupkd35VLY8AL9XSQEtY5QhhLvV5jj",
	})
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
	expect := dstest.Template(t, `Dscache:
 Dscache.Users:
  0) user=peer profileID={{ .profileID }}
 Dscache.Refs:
  0) initID        = vrh4iurbzeyx42trlddzvtoiqevmy2d3mxex4ojd4mxv7cudhlwq
     profileID     = {{ .profileID }}
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = cities_ds
     bodySize      = 155
     bodyRows      = 5
     commitTime    = 978310861
     headRef       = {{ .citiesHeadRef }}
  1) initID        = ekwzgcu4s4o4xchsoip3oa3j45ko5n7pybtizgvsbudojbhxuita
     profileID     = {{ .profileID }}
     topIndex      = 0
     cursorIndex   = 0
     prettyName    = new_ds
     commitTime    = -62135596800
     fsiPath       = /tmp/json_body
`, map[string]string{
		"profileID":     "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
		"citiesHeadRef": "/mem/QmQqnpYhA7UfcyCLkupkd35VLY8AL9XSQEtY5QhhLvV5jj",
	})

	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

// Test that `init` with no directory will use the current one
func TestInitWithCurrentDir(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	initDir := filepath.Join(run.TmpDir, "path")

	// Create a directory that will become the working directory
	if err := os.Mkdir(initDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Change to it
	if err := os.Chdir(initDir); err != nil {
		t.Fatal(err)
	}

	// Don't pass the working directory path, `init` will use the current directory
	err := run.InitWithParams(
		&InitDatasetParams{
			Name:   "new_ds",
			Format: "csv",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the directory contains the files that we expect
	dirContents := listDirectory(initDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents did not match (-want +got):\n%s", diff)
	}
}

// Test that `init` with an explicit directory will create it
func TestInitWithExplicitDir(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	initDir := filepath.Join(run.TmpDir, "path/to/dataset")

	// Pass the path for the directory, though it doesn't exist yet, `init` will make it
	err := run.InitWithParams(
		&InitDatasetParams{
			Name:      "new_ds",
			TargetDir: initDir,
			Format:    "csv",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the directory contains the files that we expect
	dirContents := listDirectory(initDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents did not match (-want +got):\n%s", diff)
	}
}

// Test that if `init` fails after requested directory has been created, that directory
// will be removed as part of rollback
func TestInitRollbackRemovesDirectory(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	// Save a dataset with one version.
	run.MustSaveFromBody(t, "movie_ds", "testdata/cities_2/body.csv")
	// Change to the temp directory to create some directory
	run.ChdirToRoot()

	// Create one directory, will run `qri init` targeting a non-existent subdirectory
	if err := os.Mkdir("path", 0644); err != nil {
		t.Fatal(err)
	}

	// Init should fail because the reference already exists in logbook
	// TODO(dustmop): Change this to some "fake" resolver, which holds references in memory,
	// and use dependency injection for the test. Remove the need to add a reference to logbook.
	err := run.InitWithParams(
		&InitDatasetParams{
			Name:      "movie_ds",
			TargetDir: filepath.Join(run.TmpDir, "path/to/dataset"),
			Format:    "csv",
		},
	)
	if err == nil {
		t.Fatal("expected error from init, did not get one")
	}

	// Directory "./path/to/" does not exist
	if _, err = os.Stat("path/to/"); err == nil {
		t.Error("directory \"./path/to\" should have been removed by rollback")
	}
	// Directory "./path/" should still exist
	if _, err = os.Stat("path"); os.IsNotExist(err) {
		t.Error("directory \"./path\" should *NOT* have been removed by rollback")
	}
}

func listDirectory(path string) []string {
	contents := []string{}
	finfos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil
	}
	for _, fi := range finfos {
		contents = append(contents, fi.Name())
	}
	sort.Strings(contents)
	return contents
}
