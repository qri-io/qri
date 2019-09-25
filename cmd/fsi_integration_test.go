package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset/dsfs"
	"github.com/spf13/cobra"
)

// FSITestRunner holds test info for fsi integration tests, for convenient cleanup.
type FSITestRunner struct {
	RepoRoot    *TestRepoRoot
	Context     context.Context
	ContextDone func()
	TsFunc      func() time.Time
	Pwd         string
	RootPath    string
	WorkPath    string
	CmdR        *cobra.Command
}

// NewFSITestRunner returns a new FSITestRunner.
func NewFSITestRunner(t *testing.T, testName string) *FSITestRunner {
	root := NewTestRepoRoot(t, testName)

	fr := FSITestRunner{}
	fr.RepoRoot = &root
	fr.Context, fr.ContextDone = context.WithCancel(context.Background())

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	fr.TsFunc = dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	var err error
	fr.Pwd, err = os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Construct a temp directory, under which any fsi linked directories will be created.
	fr.RootPath, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	fr.WorkPath = ""
	return &fr
}

// Delete cleans up after a FSITestRunner is done being used.
func (fr *FSITestRunner) Delete() {
	os.Chdir(fr.Pwd)
	if fr.WorkPath != "" {
		defer os.RemoveAll(fr.WorkPath)
	}
	defer os.RemoveAll(fr.RootPath)
	dsfs.Timestamp = fr.TsFunc
	fr.ContextDone()
	fr.RepoRoot.Delete()
}

// ExecCommand executes the given command string
func (fr *FSITestRunner) ExecCommand(cmdText string) error {
	fr.CmdR = fr.RepoRoot.CreateCommandRunner(fr.Context)
	return executeCommand(fr.CmdR, cmdText)
}

// GetCommandOutput returns the standard output from the previously executed command
func (fr *FSITestRunner) GetCommandOutput() string {
	return fr.RepoRoot.GetOutput()
}

// ChdirToRoot changes the current directory to the temporary root
func (fr *FSITestRunner) ChdirToRoot() {
	os.Chdir(fr.RootPath)
}

// ChangeToWorkDir changes to the already created working directory. Panics if it doesn't exist.
func (fr *FSITestRunner) ChdirToWorkDir(subdir string) string {
	fr.WorkPath = filepath.Join(fr.RootPath, subdir)
	if err := os.Chdir(fr.WorkPath); err != nil {
		panic(err)
	}
	return fr.WorkPath
}

// CreateAndChangeToWorkDir creates and changes to the working directory
func (fr *FSITestRunner) CreateAndChdirToWorkDir(subdir string) string {
	fr.WorkPath = filepath.Join(fr.RootPath, subdir)
	err := os.MkdirAll(fr.WorkPath, os.ModePerm)
	if err != nil {
		panic(err)
	}
	os.Chdir(fr.WorkPath)
	return fr.WorkPath
}

// Test using "init" to create a new linked directory, using status to see the added files,
// then saving to create the dataset, leading to a clean status in the directory.
func TestInitStatusSave(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_init_status_save")
	defer fr.Delete()

	workDir := fr.CreateAndChdirToWorkDir("brand_new")

	// Init as a linked directory.
	if err := fr.ExecCommand("qri init --name brand_new --format csv"); err != nil {
		t.Fatalf(err.Error())
	}

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/brand_new]

  add: meta (source: meta.json)
  add: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Save the new dataset.
	if err := fr.ExecCommand("qri save"); err != nil {
		t.Fatalf(err.Error())
	}

	// TODO: Verify that files are in ipfs repo.

	// Status again, check that the working directory is clean.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/brand_new"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test that checkout, used on a simple dataset with a body.json and no meta, creates a
// working directory with a clean status.
func TestCheckoutSimpleStatus(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_checkout_simple_status")
	defer fr.Delete()

	// Save a dataset containing a body.json, no meta, nothing special.
	err := fr.ExecCommand("qri save --body=testdata/movies/body_two.json me/two_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err := fr.ExecCommand("qri checkout me/two_movies"); err != nil {
		t.Fatalf(err.Error())
	}

	workDir := fr.ChdirToWorkDir("two_movies")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory is clean.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/two_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify the body.json file.
	modifyFileUsingStringReplace("body.json", "Avatar", "The Avengers")

	// Status again, check that the body is changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/two_movies]

  modified: body (source: body.json)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Create meta.json with a title.
	if err = ioutil.WriteFile("meta.json", []byte(`{"title": "hello"}`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Status yet again, check that the meta is added.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	expect = `for linked dataset [test_peer/two_movies]

  add: meta (source: meta.json)
  modified: body (source: body.json)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test checking out a dataset with a schema, and body.csv.
func TestCheckoutWithStructure(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_checkout_with_structure")
	defer fr.Delete()

	// Save a dataset containing a body.csv and meta.
	err := fr.ExecCommand("qri save --body=testdata/movies/body_ten.csv --file=testdata/movies/meta_override.yaml me/ten_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err = fr.ExecCommand("qri checkout me/ten_movies"); err != nil {
		t.Fatalf(err.Error())
	}

	workPath := fr.ChdirToWorkDir("ten_movies")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workPath)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory is clean.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/ten_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify the body.json file.
	modifyFileUsingStringReplace("body.csv", "Avatar", "The Avengers")

	// Status again, check that the body is changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/ten_movies]

  modified: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify meta.json by changing the title.
	if err = ioutil.WriteFile("meta.json", []byte(`{"title": "hello"}`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Status yet again, check that the meta is changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	expect = `for linked dataset [test_peer/ten_movies]

  modified: meta (source: meta.json)
  modified: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Remove meta.json.
	if err = os.Remove("meta.json"); err != nil {
		t.Fatalf(err.Error())
	}

	// Status one last time, check that the meta was removed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	expect = `for linked dataset [test_peer/ten_movies]

  removed:  meta
  modified: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test checkout and modifying structure & schema, then checking status.
func TestCheckoutAndModifyStructure(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_checkout_and_modify_schema")
	defer fr.Delete()

	// Save a dataset containing a body.csv, no meta, nothing special.
	err := fr.ExecCommand("qri save --body=testdata/movies/body_ten.csv me/more_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err = fr.ExecCommand("qri checkout me/more_movies"); err != nil {
		t.Fatalf(err.Error())
	}

	workPath := fr.ChdirToWorkDir("more_movies")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workPath)
	expectContents := []string{".qri-ref", "body.csv", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory is clean.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/more_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Create schema.json with a minimal schema.
	if err = ioutil.WriteFile("structure.json", []byte(`{ "format": "csv", "schema": {"type": "array"}}`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Status again, check that the body is changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/more_movies]

  modified: structure (source: structure.json)
  modified: schema (source: structure.json)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test that status displays parse errors correctly
func TestStatusParseError(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_status_parse_error")
	defer fr.Delete()

	// Save a dataset containing a body.json and meta component
	err := fr.ExecCommand("qri save --body=testdata/movies/body_two.json --file=testdata/movies/meta_override.yaml me/bad_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Change to a temporary directory.
	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err = fr.ExecCommand("qri checkout me/bad_movies"); err != nil {
		t.Fatalf(err.Error())
	}

	_ = fr.ChdirToWorkDir("bad_movies")

	// Modify the meta.json so that it fails to parse.
	if err = ioutil.WriteFile("meta.json", []byte(`{"title": "hello}`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Status, check that status shows the parse error.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/bad_movies]

  parse error: meta (source: meta.json)

fix these problems before saving this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test that status displays parse errors even for the body component
func TestBodyParseError(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_status_parse_error")
	defer fr.Delete()

	// Save a dataset containing a body.json and meta component
	err := fr.ExecCommand("qri save --body=testdata/movies/body_two.json --file=testdata/movies/meta_override.yaml me/bad_body")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Change to a temporary directory.
	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err = fr.ExecCommand("qri checkout me/bad_body"); err != nil {
		t.Fatalf(err.Error())
	}

	_ = fr.ChdirToWorkDir("bad_body")

	// Modify the meta.json so that it fails to parse.
	if err = ioutil.WriteFile("body.json", []byte(`{"title": "hello}`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Status, check that status shows the parse error.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/bad_body]

  parse error: body (source: body.json)

fix these problems before saving this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test that parse errors are also properly shown for schema.
func TestStatusParseErrorForSchema(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_status_parse_error_for_schema")
	defer fr.Delete()

	// Save a dataset containing a body.json and meta component
	err := fr.ExecCommand("qri save --body=testdata/movies/body_ten.csv me/ten_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err = fr.ExecCommand("qri checkout me/ten_movies"); err != nil {
		t.Fatalf(err.Error())
	}

	_ = fr.ChdirToWorkDir("ten_movies")

	// Modify the meta.json so that it fails to parse.
	if err = ioutil.WriteFile("schema.json", []byte(`{"type":`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Status, check that status shows the parse error.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/ten_movies]

  parse error: schema (source: schema.json)

fix these problems before saving this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test status at specific versions
func TestStatusAtVersion(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_status_at_version")
	defer fr.Delete()

	// First version has only a body
	err := fr.ExecCommand("qri save --body=testdata/movies/body_two.json me/status_ver")
	if err != nil {
		t.Fatalf(err.Error())
	}
	ref1 := parseRefFromSave(fr.GetCommandOutput())

	// Add a meta
	err = fr.ExecCommand("qri save --file=testdata/movies/meta_override.yaml me/status_ver")
	if err != nil {
		t.Fatalf(err.Error())
	}
	ref2 := parseRefFromSave(fr.GetCommandOutput())

	// Change the meta
	err = fr.ExecCommand("qri save --file=testdata/movies/meta_another.yaml me/status_ver")
	if err != nil {
		t.Fatalf(err.Error())
	}
	ref3 := parseRefFromSave(fr.GetCommandOutput())

	// Change the body
	err = fr.ExecCommand("qri save --body=testdata/movies/body_four.json me/status_ver")
	if err != nil {
		t.Fatalf(err.Error())
	}
	ref4 := parseRefFromSave(fr.GetCommandOutput())

	// Status for the first version of the dataset, both body and schema were added.
	if err = fr.ExecCommand(fmt.Sprintf("qri status %s", ref1)); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	expect := `  schema: add
  body: add
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Status for the second version, meta added.
	if err = fr.ExecCommand(fmt.Sprintf("qri status %s", ref2)); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	expect = `  meta: add
  schema: unmodified
  body: unmodified
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Status for the third version, meta modified.
	if err = fr.ExecCommand(fmt.Sprintf("qri status %s", ref3)); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	expect = `  meta: modified
  schema: unmodified
  body: unmodified
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Status for the fourth version, body modified.
	if err = fr.ExecCommand(fmt.Sprintf("qri status %s", ref4)); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	expect = `  meta: unmodified
  schema: unmodified
  body: modified
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test checking out, modifying components, then using restore to undo the modification.
func TestCheckoutAndRestore(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_checkout_and_restore")
	defer fr.Delete()

	// Save a dataset containing a body.csv and meta.
	err := fr.ExecCommand("qri save --body=testdata/movies/body_ten.csv --file=testdata/movies/meta_override.yaml me/ten_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err = fr.ExecCommand("qri checkout me/ten_movies"); err != nil {
		t.Fatalf(err.Error())
	}

	_ = fr.ChdirToWorkDir("ten_movies")

	// Modify meta.json by changing the title.
	if err = ioutil.WriteFile("meta.json", []byte(`{"title": "hello"}`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Status to check that the meta is changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/ten_movies]

  modified: meta (source: meta.json)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Restore to get the old meta back.
	if err = fr.ExecCommand("qri restore meta"); err != nil {
		t.Fatalf(err.Error())
	}

	// Status again, to validate that meta is no longer changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/ten_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify struture.json
	if err = ioutil.WriteFile("structure.json", []byte(`{ "format" : "csv", "schema": {"type": "array"}}`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Status to check that the schema is changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	expect = `for linked dataset [test_peer/ten_movies]

  modified: structure (source: structure.json)
  modified: schema (source: structure.json)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Restore to get the old schema back.
	if err = fr.ExecCommand("qri restore schema"); err != nil {
		t.Fatalf(err.Error())
	}

	// Status again, to validate that schema is no longer changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output = fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/ten_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test restoring previous version
func TestRestorePreviousVersion(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_restore_prev_version")
	defer fr.Delete()

	// First version has only a body
	err := fr.ExecCommand("qri save --body=testdata/movies/body_two.json me/prev_ver")
	if err != nil {
		t.Fatalf(err.Error())
	}
	_ = parseRefFromSave(fr.GetCommandOutput())

	// Add a meta
	err = fr.ExecCommand("qri save --file=testdata/movies/meta_override.yaml me/prev_ver")
	if err != nil {
		t.Fatalf(err.Error())
	}
	ref2 := parseRefFromSave(fr.GetCommandOutput())

	// Change the meta
	err = fr.ExecCommand("qri save --file=testdata/movies/meta_another.yaml me/prev_ver")
	if err != nil {
		t.Fatalf(err.Error())
	}
	_ = parseRefFromSave(fr.GetCommandOutput())

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err := fr.ExecCommand("qri checkout me/prev_ver"); err != nil {
		t.Fatalf(err.Error())
	}

	_ = fr.ChdirToWorkDir("prev_ver")

	// Verify that the status is clean
	if err = fr.ExecCommand(fmt.Sprintf("qri status")); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/prev_ver"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Read meta.json, contains the contents of meta_another.yaml
	metaContents, err := ioutil.ReadFile("meta.json")
	if err != nil {
		t.Fatalf(err.Error())
	}
	expectContents := "{\n \"qri\": \"md:0\",\n \"title\": \"yet another title\"\n}"
	if diff := cmp.Diff(expectContents, string(metaContents)); diff != "" {
		t.Errorf("meta.json contents (-want +got):\n%s", diff)
	}

	// TODO(dlong): Handle full dataset paths, including peername and dataset name.

	pos := strings.Index(ref2, "/ipfs/")
	path := ref2[pos:]

	// Restore the previous version
	if err = fr.ExecCommand(fmt.Sprintf("qri restore %s", path)); err != nil {
		t.Fatalf(err.Error())
	}

	// Read meta.json, due to restore, it has the old data from meta_override.yaml
	metaContents, err = ioutil.ReadFile("meta.json")
	if err != nil {
		t.Fatalf(err.Error())
	}
	expectContents = "{\n \"qri\": \"md:0\",\n \"title\": \"different title\"\n}"
	if diff := cmp.Diff(expectContents, string(metaContents)); diff != "" {
		t.Errorf("meta.json contents (-want +got):\n%s", diff)
	}
}

// Test that restore deletes a component that didn't exist before
func TestRestoreDeleteComponent(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_restore_delete_component")
	defer fr.Delete()

	// First version has only a body
	err := fr.ExecCommand("qri save --body=testdata/movies/body_ten.csv me/del_cmp")
	if err != nil {
		t.Fatalf(err.Error())
	}
	_ = parseRefFromSave(fr.GetCommandOutput())

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err := fr.ExecCommand("qri checkout me/del_cmp"); err != nil {
		t.Fatalf(err.Error())
	}

	workDir := fr.ChdirToWorkDir("del_cmp")

	// Modify the body.json file.
	modifyFileUsingStringReplace("body.csv", "Avatar", "The Avengers")

	// Modify meta.json by changing the title.
	if err = ioutil.WriteFile("meta.json", []byte(`{"title": "hello"}`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Restore to erase the meta component.
	if err := fr.ExecCommand("qri restore meta"); err != nil {
		t.Fatalf(err.Error())
	}

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/del_cmp]

  modified: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test that restore deletes a component if there was no previous version
func TestRestoreWithNoHistory(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_restore_no_history")
	defer fr.Delete()

	workDir := fr.CreateAndChdirToWorkDir("new_folder")

	// Init as a linked directory.
	if err := fr.ExecCommand("qri init --name new_folder --format csv"); err != nil {
		t.Fatalf(err.Error())
	}

	// Restore to get erase the meta component.
	if err := fr.ExecCommand("qri restore meta"); err != nil {
		t.Fatalf(err.Error())
	}

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/new_folder]

  add: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test using "init" with a source body path
func TestInitWithSourceBodyPath(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_init_source_body_path")
	defer fr.Delete()

	sourceFile, err := filepath.Abs("testdata/days_of_week.csv")
	if err != nil {
		panic(err)
	}

	workDir := fr.CreateAndChdirToWorkDir("init_source")

	// Init with a source body path.
	if err := fr.ExecCommand(fmt.Sprintf("qri init --name init_source --source-body-path %s", sourceFile)); err != nil {
		t.Fatalf(err.Error())
	}

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/init_source]

  add: meta (source: meta.json)
  add: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Read body.csv
	actualBody, err := ioutil.ReadFile("body.csv")
	if err != nil {
		t.Fatalf(err.Error())
	}
	expectBody, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("meta.json contents (-want +got):\n%s", diff)
	}
}

// Test init with a directory will create that directory
func TestInitWithDirectory(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_init_with_directory")
	defer fr.Delete()

	fr.ChdirToRoot()

	// Init with a directory to create.
	if err := fr.ExecCommand(fmt.Sprintf("qri init init_dir --name init_dir --format csv")); err != nil {
		t.Fatalf(err.Error())
	}

	// Directory has been created by `qri init`
	workDir := fr.ChdirToWorkDir("init_dir")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatalf(err.Error())
	}

	output := fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/init_dir]

  add: meta (source: meta.json)
  add: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

func parseRefFromSave(output string) string {
	pos := strings.Index(output, "saved: ")
	ref := output[pos+7:]
	return strings.TrimSpace(ref)
}

func cmpTextLines(left, right string) string {
	lside := strings.Split(left, "\n")
	rside := strings.Split(right, "\n")
	return cmp.Diff(lside, rside)
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

func modifyFileUsingStringReplace(filename, find, replace string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	text := string(data)
	text = strings.Replace(text, find, replace, -1)
	err = ioutil.WriteFile(filename, []byte(text), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func cleanStatusMessage(dsref string) string {
	template := `for linked dataset [%s]

working directory clean
`
	return fmt.Sprintf(template, dsref)
}
