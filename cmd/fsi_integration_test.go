package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/fsi"
)

// FSITestRunner holds test info for fsi integration tests, for convenient cleanup.
type FSITestRunner struct {
	TestRunner
	Pwd      string
	RootPath string
	WorkPath string
}

// NewFSITestRunner returns a new FSITestRunner.
func NewFSITestRunner(t *testing.T, testName string) *FSITestRunner {
	inner := NewTestRunner(t, "test_peer", testName)
	return newFSITestRunnerFromInner(t, inner)
}

// NewFSITestRunnerWithMockRemoteClient returns a new FSITestRunner.
func NewFSITestRunnerWithMockRemoteClient(t *testing.T, testName string) *FSITestRunner {
	inner := NewTestRunnerWithMockRemoteClient(t, "test_peer", testName)
	return newFSITestRunnerFromInner(t, inner)
}

func newFSITestRunnerFromInner(t *testing.T, inner *TestRunner) *FSITestRunner {
	run := FSITestRunner{TestRunner: *inner}

	var err error
	run.Pwd, err = os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Construct a temp directory, under which any fsi linked directories will be created.
	run.RootPath, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	run.WorkPath = ""

	run.Teardown = func() {
		os.Chdir(run.Pwd)
		os.RemoveAll(run.RootPath)
	}

	return &run
}

// ChdirToRoot changes the current directory to the temporary root
func (run *FSITestRunner) ChdirToRoot() {
	os.Chdir(run.RootPath)
}

// ChangeToWorkDir changes to the already created working directory. Panics if it doesn't exist.
func (run *FSITestRunner) ChdirToWorkDir(subdir string) string {
	run.WorkPath = filepath.Join(run.RootPath, subdir)
	if err := os.Chdir(run.WorkPath); err != nil {
		panic(err)
	}
	return run.WorkPath
}

// CreateAndChangeToWorkDir creates and changes to the working directory
func (run *FSITestRunner) CreateAndChdirToWorkDir(subdir string) string {
	run.WorkPath = filepath.Join(run.RootPath, subdir)
	err := os.MkdirAll(run.WorkPath, os.ModePerm)
	if err != nil {
		panic(err)
	}
	os.Chdir(run.WorkPath)
	return run.WorkPath
}

// Test using "init" to create a new linked directory, using status to see the added files,
// then saving to create the dataset, leading to a clean status in the directory.
func TestInitStatusSave(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_init_status_save")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("brand_new")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name brand_new --format csv")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Verify contents of the structure, there should not be a schema.
	expectText := `{
 "format": "csv",
 "qri": "st:0"
}`
	structureText := run.MustReadFile(t, filepath.Join(workDir, "structure.json"))
	if diff := cmp.Diff(expectText, structureText); diff != "" {
		t.Errorf("structure.json contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	output := run.MustExec(t, "qri status")
	expect := `for linked dataset [test_peer/brand_new]

  add: meta (source: meta.json)
  add: structure (source: structure.json)
  add: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Save the new dataset.
	run.MustExec(t, "qri save")

	// TODO: Verify that files are in ipfs repo.

	// Verify that the .qri-ref contains the full path for the saved dataset.
	contents := run.MustReadFile(t, ".qri-ref")
	// TODO(dlong): Fix me, should write the updated FSI link with the dsref head
	expect = "test_peer/brand_new"
	if diff := cmp.Diff(expect, contents); diff != "" {
		t.Errorf(".qri-ref contents (-want +got):\n%s", diff)
	}

	// Status again, check that the working directory is clean.
	output = run.MustExec(t, "qri status")
	if diff := cmpTextLines(cleanStatusMessage("test_peer/brand_new"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test init command can use an explicit directory
func TestInitExplicitDirectory(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_init_explicit_dir")
	defer run.Delete()

	run.ChdirToRoot()
	run.MustExec(t, "qri init --name explicit_dir --format csv explicit_dir")
	workDir := filepath.Join(run.RootPath, "explicit_dir")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}
}

// Test that we can get the body even if structure has been deleted.
func TestGetBodyWithoutStructure(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_get_body_without_structure")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("body_only")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name body_only --format csv")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Remove the structure.
	if err := os.Remove(filepath.Join(workDir, "structure.json")); err != nil {
		t.Fatal(err)
	}

	// Get the body, even though there's no structure. One will be inferred.
	output := run.MustExec(t, "qri get body")
	expectBody := "for linked dataset [test_peer/body_only]\n\none,two,3\nfour,five,6\n\n"
	if diff := cmp.Diff(expectBody, output); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}
}

// Test init command can create a json body using the format flag
func TestInitForJsonBody(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_init_json_body")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("json_body")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name json_body --format json")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.json", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}
}

// Test init command can create a json body from a source body
func TestInitWithJsonSourceBodyPath(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_init_json_body")
	defer run.Delete()

	sourceFile, err := filepath.Abs("testdata/movies/body_four.json")
	if err != nil {
		panic(err)
	}

	workDir := run.CreateAndChdirToWorkDir("json_body")

	// Init as a linked directory.
	run.MustExec(t, fmt.Sprintf("qri init --name json_body --source-body-path %s", sourceFile))

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.json", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}
}

// Test that checkout, used on a simple dataset with a body.json and no meta, creates a
// working directory with a clean status.
func TestCheckoutSimpleStatus(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_checkout_simple_status")
	defer run.Delete()

	// Save a dataset containing a body.json, no meta, nothing special.
	run.MustExec(t, "qri save --body=testdata/movies/body_two.json me/two_movies")

	run.ChdirToRoot()

	// Checkout the newly created dataset.
	run.MustExec(t, "qri checkout me/two_movies")

	workDir := run.ChdirToWorkDir("two_movies")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory is clean.
	output := run.MustExec(t, "qri status")
	if diff := cmpTextLines(cleanStatusMessage("test_peer/two_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify the body.json file.
	modifyFileUsingStringReplace("body.json", "Avatar", "The Avengers")

	// Status again, check that the body is changed.
	output = run.MustExec(t, "qri status")
	expect := `for linked dataset [test_peer/two_movies]

  modified: body (source: body.json)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Create meta.json with a title.
	run.MustWriteFile(t, "meta.json", `{"title": "hello"}`)

	// Status yet again, check that the meta is added.
	output = run.MustExec(t, "qri status")
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
	run := NewFSITestRunner(t, "qri_test_checkout_with_structure")
	defer run.Delete()

	// Save a dataset containing a body.csv and meta.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv --file=testdata/movies/meta_override.yaml me/ten_movies")

	run.ChdirToRoot()

	// Checkout the newly created dataset.
	run.MustExec(t, "qri checkout me/ten_movies")

	workPath := run.ChdirToWorkDir("ten_movies")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workPath)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory is clean.
	output := run.MustExec(t, "qri status")
	if diff := cmpTextLines(cleanStatusMessage("test_peer/ten_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify the body.json file.
	modifyFileUsingStringReplace("body.csv", "Avatar", "The Avengers")

	// Status again, check that the body is changed.
	run.MustExec(t, "qri status")

	output = run.GetCommandOutput()
	expect := `for linked dataset [test_peer/ten_movies]

  modified: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify meta.json by changing the title.
	run.MustWriteFile(t, "meta.json", `{"title": "hello"}`)

	// Status yet again, check that the meta is changed.
	output = run.MustExec(t, "qri status")
	expect = `for linked dataset [test_peer/ten_movies]

  modified: meta (source: meta.json)
  modified: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Remove meta.json.
	if err := os.Remove("meta.json"); err != nil {
		t.Fatal(err)
	}

	// Status one last time, check that the meta was removed.
	output = run.MustExec(t, "qri status")
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
	run := NewFSITestRunner(t, "qri_test_checkout_and_modify_schema")
	defer run.Delete()

	// Save a dataset containing a body.csv, no meta, nothing special.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/more_movies")

	run.ChdirToRoot()

	// Checkout the newly created dataset.
	run.MustExec(t, "qri checkout me/more_movies")

	workPath := run.ChdirToWorkDir("more_movies")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workPath)
	expectContents := []string{".qri-ref", "body.csv", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory is clean.
	output := run.MustExec(t, "qri status")
	if diff := cmpTextLines(cleanStatusMessage("test_peer/more_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Create structure.json with a minimal schema.
	run.MustWriteFile(t, "structure.json", `{ "format": "csv", "schema": {"type": "array"}}`)

	// Status again, check that the body is changed.
	output = run.MustExec(t, "qri status")
	expect := `for linked dataset [test_peer/more_movies]

  modified: structure (source: structure.json)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test that status displays parse errors correctly
func TestStatusParseError(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_status_parse_error")
	defer run.Delete()

	// Save a dataset containing a body.json and meta component
	run.MustExec(t, "qri save --body=testdata/movies/body_two.json --file=testdata/movies/meta_override.yaml me/bad_movies")

	// Change to a temporary directory.
	run.ChdirToRoot()

	// Checkout the newly created dataset.
	run.MustExec(t, "qri checkout me/bad_movies")

	_ = run.ChdirToWorkDir("bad_movies")

	// Modify the meta.json so that it fails to parse.
	run.MustWriteFile(t, "meta.json", `{"title": "hello}`)

	// Status, check that status shows the parse error.
	output := run.MustExec(t, "qri status")
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
	run := NewFSITestRunner(t, "qri_test_status_parse_error")
	defer run.Delete()

	// Save a dataset containing a body.json and meta component
	run.MustExec(t, "qri save --body=testdata/movies/body_two.json --file=testdata/movies/meta_override.yaml me/bad_body")

	// Change to a temporary directory.
	run.ChdirToRoot()

	// Checkout the newly created dataset.
	run.MustExec(t, "qri checkout me/bad_body")

	_ = run.ChdirToWorkDir("bad_body")

	// Modify the meta.json so that it fails to parse.
	run.MustWriteFile(t, "body.json", `{"title": "hello}`)

	// Status, check that status shows the parse error.
	output := run.MustExec(t, "qri status")
	expect := `for linked dataset [test_peer/bad_body]

  parse error: body (source: body.json)

fix these problems before saving this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test that parse errors are also properly shown for structure.
func TestStatusParseErrorForStructure(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_status_parse_error_for_structure")
	defer run.Delete()

	// Save a dataset containing a body.json and meta component
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/ten_movies")

	run.ChdirToRoot()

	// Checkout the newly created dataset.
	run.MustExec(t, "qri checkout me/ten_movies")

	_ = run.ChdirToWorkDir("ten_movies")

	// Modify the meta.json so that it fails to parse.
	run.MustWriteFile(t, "structure.json", `{"format":`)

	// Status, check that status shows the parse error.
	output := run.MustExec(t, "qri status")
	expect := `for linked dataset [test_peer/ten_movies]

  parse error: structure (source: structure.json)

fix these problems before saving this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test status at specific versions
func TestStatusAtVersion(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_status_at_version")
	defer run.Delete()

	// First version has only a body
	output := run.MustExec(t, "qri save --body=testdata/movies/body_two.json me/status_ver")
	ref1 := parseRefFromSave(output)

	// Add a meta
	output = run.MustExec(t, "qri save --file=testdata/movies/meta_override.yaml me/status_ver")
	ref2 := parseRefFromSave(output)

	// Change the meta
	output = run.MustExec(t, "qri save --file=testdata/movies/meta_another.yaml me/status_ver")
	ref3 := parseRefFromSave(output)

	// Change the body
	output = run.MustExec(t, "qri save --body=testdata/movies/body_four.json me/status_ver")
	ref4 := parseRefFromSave(output)

	// Status for the first version of the dataset, both body and schema were added.
	output = run.MustExec(t, fmt.Sprintf("qri status %s", ref1))
	expect := `  structure: add
  body: add
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Status for the second version, meta added.
	output = run.MustExec(t, fmt.Sprintf("qri status %s", ref2))
	expect = `  meta: add
  structure: unmodified
  body: unmodified
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Status for the third version, meta modified.
	output = run.MustExec(t, fmt.Sprintf("qri status %s", ref3))
	expect = `  meta: modified
  structure: unmodified
  body: unmodified
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Status for the fourth version, body modified.
	output = run.MustExec(t, fmt.Sprintf("qri status %s", ref4))
	expect = `  meta: unmodified
  structure: unmodified
  body: modified
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test checking out, modifying components, then using restore to undo the modification.
func TestCheckoutAndRestore(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_checkout_and_restore")
	defer run.Delete()

	// Save a dataset containing a body.csv and meta.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv --file=testdata/movies/meta_override.yaml me/ten_movies")

	run.ChdirToRoot()

	// Checkout the newly created dataset.
	run.MustExec(t, "qri checkout me/ten_movies")

	_ = run.ChdirToWorkDir("ten_movies")

	// Modify meta.json by changing the title.
	run.MustWriteFile(t, "meta.json", `{"title": "hello"}`)

	// Status to check that the meta is changed.
	output := run.MustExec(t, "qri status")
	expect := `for linked dataset [test_peer/ten_movies]

  modified: meta (source: meta.json)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Restore to get the old meta back.
	run.MustExec(t, "qri restore meta")

	// Status again, to validate that meta is no longer changed.
	output = run.MustExec(t, "qri status")
	if diff := cmpTextLines(cleanStatusMessage("test_peer/ten_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify struture.json
	run.MustWriteFile(t, "structure.json", `{ "format" : "csv", "schema": {"type": "array"}}`)

	// Status to check that the schema is changed.
	output = run.MustExec(t, "qri status")
	expect = `for linked dataset [test_peer/ten_movies]

  modified: structure (source: structure.json)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Restore to get the old schema back.
	run.MustExec(t, "qri restore structure")

	// Status again, to validate that schema is no longer changed.
	output = run.MustExec(t, "qri status")
	if diff := cmpTextLines(cleanStatusMessage("test_peer/ten_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test restoring previous version
func TestRestorePreviousVersion(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_restore_prev_version")
	defer run.Delete()

	// First version has only a body
	output := run.MustExec(t, "qri save --body=testdata/movies/body_two.json me/prev_ver")
	_ = parseRefFromSave(output)

	// Add a meta
	output = run.MustExec(t, "qri save --file=testdata/movies/meta_override.yaml me/prev_ver")
	ref2 := parseRefFromSave(output)

	// Change the meta
	output = run.MustExec(t, "qri save --file=testdata/movies/meta_another.yaml me/prev_ver")
	_ = parseRefFromSave(output)

	run.ChdirToRoot()

	// Checkout the newly created dataset.
	run.MustExec(t, "qri checkout me/prev_ver")

	_ = run.ChdirToWorkDir("prev_ver")

	// Verify that the status is clean
	output = run.MustExec(t, "qri status")
	if diff := cmpTextLines(cleanStatusMessage("test_peer/prev_ver"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Read meta.json, contains the contents of meta_another.yaml
	metaContents := run.MustReadFile(t, "meta.json")
	expectContents := "{\n \"qri\": \"md:0\",\n \"title\": \"yet another title\"\n}"
	if diff := cmp.Diff(expectContents, metaContents); diff != "" {
		t.Errorf("meta.json contents (-want +got):\n%s", diff)
	}

	// TODO(dlong): Handle full dataset paths, including peername and dataset name.

	pos := strings.Index(ref2, "/ipfs/")
	path := ref2[pos:]

	// Restore the previous version
	run.MustExec(t, fmt.Sprintf("qri restore %s", path))

	// Read meta.json, due to restore, it has the old data from meta_override.yaml
	metaContents = run.MustReadFile(t, "meta.json")
	expectContents = "{\n \"qri\": \"md:0\",\n \"title\": \"different title\"\n}"
	if diff := cmp.Diff(expectContents, metaContents); diff != "" {
		t.Errorf("meta.json contents (-want +got):\n%s", diff)
	}
}

// Test that restore deletes a component that didn't exist before
func TestRestoreDeleteComponent(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_restore_delete_component")
	defer run.Delete()

	// First version has only a body
	output := run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/del_cmp")
	_ = parseRefFromSave(output)

	run.ChdirToRoot()

	// Checkout the newly created dataset.
	run.MustExec(t, "qri checkout me/del_cmp")

	workDir := run.ChdirToWorkDir("del_cmp")

	// Modify the body.json file.
	modifyFileUsingStringReplace("body.csv", "Avatar", "The Avengers")

	// Modify meta.json by changing the title.
	run.MustWriteFile(t, "meta.json", `{"title": "hello"}`)

	// Restore to erase the meta component.
	run.MustExec(t, "qri restore meta")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	output = run.MustExec(t, "qri status")
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
	run := NewFSITestRunner(t, "qri_test_restore_no_history")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("new_folder")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name new_folder --format csv")

	// Restore to get erase the meta component.
	run.MustExec(t, "qri restore meta")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	output := run.MustExec(t, "qri status")
	expect := `for linked dataset [test_peer/new_folder]

  add: structure (source: structure.json)
  add: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test creating a readme and then rendering it.
func TestRenderReadme(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_render_readme")
	defer run.Delete()

	_ = run.CreateAndChdirToWorkDir("render_readme")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name render_readme --format csv")

	// Create readme.md with some text.
	run.MustWriteFile(t, "readme.md", "# hi\nhello\n")

	// Status, check that the working directory has added files including readme.
	output := run.MustExec(t, "qri status")
	expect := `for linked dataset [test_peer/render_readme]

  add: meta (source: meta.json)
  add: structure (source: structure.json)
  add: readme (source: readme.md)
  add: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Save the new dataset.
	run.MustExec(t, "qri save")

	// Status again, check that the working directory is clean.
	output = run.MustExec(t, "qri status")
	if diff := cmpTextLines(cleanStatusMessage("test_peer/render_readme"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Render the readme, check the html.
	output = run.MustExec(t, "qri render")
	expectBody := `for linked dataset [test_peer/render_readme]

<h1>hi</h1>

<p>hello</p>
`
	if diff := cmp.Diff(expectBody, output); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}
}

// Test using "init" with a source body path
func TestInitWithSourceBodyPath(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_init_source_body_path")
	defer run.Delete()

	sourceFile, err := filepath.Abs("testdata/days_of_week.csv")
	if err != nil {
		panic(err)
	}

	workDir := run.CreateAndChdirToWorkDir("init_source")

	// Init with a source body path.
	run.MustExec(t, fmt.Sprintf("qri init --name init_source --source-body-path %s", sourceFile))

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Verify contents of the structure, should have schema.
	expectText := `{
 "format": "csv",
 "formatConfig": {
  "headerRow": true,
  "lazyQuotes": true
 },
 "qri": "st:0",
 "schema": {
  "items": {
   "items": [
    {
     "title": "english",
     "type": "string"
    },
    {
     "title": "spanish",
     "type": "string"
    }
   ],
   "type": "array"
  },
  "type": "array"
 }
}`
	structureText := run.MustReadFile(t, filepath.Join(workDir, "structure.json"))
	if diff := cmp.Diff(expectText, structureText); diff != "" {
		t.Errorf("structure.json contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	output := run.MustExec(t, "qri status")
	expect := `for linked dataset [test_peer/init_source]

  add: meta (source: meta.json)
  add: structure (source: structure.json)
  add: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Read body.csv
	actualBody := run.MustReadFile(t, "body.csv")
	// TODO(dlong): Fix this test, figure out why lazyQuotes is not detected to be true.
	expectBody := `english,spanish
Sunday," domingo"
Monday," lunes"
Tuesday," martes"
Wednesday," miércoles"
Thursday," jueves"
Friday," viernes"
Saturdy," sábado"
`
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("meta.json contents (-want +got):\n%s", diff)
	}
}

// Test init with a directory will create that directory
func TestInitWithDirectory(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_init_with_directory")
	defer run.Delete()

	run.ChdirToRoot()

	// Init with a directory to create.
	run.MustExec(t, fmt.Sprintf("qri init init_dir --name init_dir --format csv"))

	// Directory has been created by `qri init`
	workDir := run.ChdirToWorkDir("init_dir")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	output := run.MustExec(t, "qri status")
	expect := `for linked dataset [test_peer/init_dir]

  add: meta (source: meta.json)
  add: structure (source: structure.json)
  add: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test making changes, then using diff to see those changes
func TestDiffAfterChange(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_diff_after_change")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("diff_change")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name diff_change --format csv")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Save the new dataset.
	run.MustExec(t, "qri save")

	// Modify meta.json with a title.
	run.MustWriteFile(t, "meta.json", `{"title": "hello"}`)

	// Modify body.csv.
	run.MustWriteFile(t, "body.csv", `lucky,number,17
four,five,321
`)

	// Status to see changes
	output := run.MustExec(t, "qri status")
	expect := `for linked dataset [test_peer/diff_change]

  modified: meta (source: meta.json)
  modified: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Diff to see changes
	output = run.MustExec(t, "qri diff")
	expect = `for linked dataset [test_peer/diff_change]

+1 element. 1 insert. 0 deletes. 4 updates.

body:
  0:
    ~ 0: "lucky"
    ~ 1: "number"
    ~ 2: 17
  1:
    ~ 2: 321
meta:
  + title: "hello"
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test that diff before save leads to a reasonable error message
func TestDiffBeforeSave(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_diff_before_save")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("diff_before")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name diff_change --format csv")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Diff should return the expected error message
	err := run.ExecCommand("qri diff")
	if err == nil {
		t.Fatal("expected error trying to init, did not get an error")
	}
	expect := `dataset has no versions, nothing to diff against`
	if err.Error() != expect {
		t.Errorf("error mismatch, expect: %s, got: %s", expect, err.Error())
	}
}

// Test that if the meta component fails to write, init will rollback
func TestInitMetaFailsToWrite(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_init_meta_fail")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("meta_fail")

	// Set the meta component to fail when trying to write it to the filesystem
	fsi.PrepareToWrite = func(c component.Component) {
		metaComp := c.Base().GetSubcomponent("meta")
		if metaComp != nil {
			meta := metaComp.(*component.MetaComponent)
			meta.DisableSerialization = true
		}
	}
	defer func() {
		fsi.PrepareToWrite = func(c component.Component) {}
	}()

	// Init as a linked directory.
	err := run.ExecCommand("qri init --name meta_fail --format csv")
	if err == nil {
		t.Fatal("expected error trying to init, did not get an error")
	}
	expect := `serialization is disabled`
	if err.Error() != expect {
		t.Errorf("error mismatch, expect: %s, got: %s", expect, err.Error())
	}

	// Verify the directory contains no files, since it rolled back.
	dirContents := listDirectory(workDir)
	expectContents := []string{}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Init with an explicit child directory.
	err = run.ExecCommand("qri init --name meta_fail --format csv subdir")
	if err == nil {
		t.Fatal("expected error trying to init, did not get an error")
	}

	// Verify that the sub-directory does not exist.
	_, err = os.Stat(filepath.Join(workDir, "subdir"))
	if !os.IsNotExist(err) {
		t.Errorf("expected \"subdir\" not to exist")
	}
}

// Test that if source-body-path doesn't exist, init will rollback
func TestInitSourceBodyPathDoesNotExist(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_init_source_not_found")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("source_not_found")

	// Init as a linked directory.
	err := run.ExecCommand("qri init --name source_not_found --source-body-path not_found.json")
	if err == nil {
		t.Fatal("expected error trying to init, did not get an error")
	}
	expect := `open not_found.json: no such file or directory`
	if err.Error() != expect {
		t.Errorf("error mismatch, expect: %s, got: %s", expect, err.Error())
	}

	// Verify the directory contains no files, since it rolled back.
	dirContents := listDirectory(workDir)
	expectContents := []string{}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
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
