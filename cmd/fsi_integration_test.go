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
	run := FSITestRunner{
		TestRunner: *NewTestRunner(t, "test_peer", testName),
	}

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
		t.Fatal(err)
	}

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
	structureText, err := ioutil.ReadFile(filepath.Join(workDir, "structure.json"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectText, string(structureText)); diff != "" {
		t.Errorf("structure.json contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
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
	if err := fr.ExecCommand("qri save"); err != nil {
		t.Fatal(err)
	}

	// TODO: Verify that files are in ipfs repo.

	// Verify that the .qri-ref contains the full path for the saved dataset.
	bytes, err := ioutil.ReadFile(".qri-ref")
	if err != nil {
		t.Fatal(err)
	}
	// TODO(dlong): Fix me, should write the updated FSI link with the dsref head
	expect = "test_peer/brand_new"
	if diff := cmp.Diff(expect, string(bytes)); diff != "" {
		t.Errorf(".qri-ref contents (-want +got):\n%s", diff)
	}

	// Status again, check that the working directory is clean.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
	}

	output = fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/brand_new"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test that we can get the body even if structure has been deleted.
func TestGetBodyWithoutStructure(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_get_body_without_structure")
	defer fr.Delete()

	workDir := fr.CreateAndChdirToWorkDir("body_only")

	// Init as a linked directory.
	if err := fr.ExecCommand("qri init --name body_only --format csv"); err != nil {
		t.Fatal(err)
	}

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
	if err := fr.ExecCommand("qri get body"); err != nil {
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
	expectBody := "for linked dataset [test_peer/body_only]\n\none,two,3\nfour,five,6\n\n"
	if diff := cmp.Diff(expectBody, output); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
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
		t.Fatal(err)
	}

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err := fr.ExecCommand("qri checkout me/two_movies"); err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/two_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify the body.json file.
	modifyFileUsingStringReplace("body.json", "Avatar", "The Avengers")

	// Status again, check that the body is changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	// Status yet again, check that the meta is added.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err = fr.ExecCommand("qri checkout me/ten_movies"); err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/ten_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify the body.json file.
	modifyFileUsingStringReplace("body.csv", "Avatar", "The Avengers")

	// Status again, check that the body is changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	// Status yet again, check that the meta is changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	// Status one last time, check that the meta was removed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err = fr.ExecCommand("qri checkout me/more_movies"); err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/more_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Create structure.json with a minimal schema.
	if err = ioutil.WriteFile("structure.json", []byte(`{ "format": "csv", "schema": {"type": "array"}}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Status again, check that the body is changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
	}

	output = fr.GetCommandOutput()
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
	fr := NewFSITestRunner(t, "qri_test_status_parse_error")
	defer fr.Delete()

	// Save a dataset containing a body.json and meta component
	err := fr.ExecCommand("qri save --body=testdata/movies/body_two.json --file=testdata/movies/meta_override.yaml me/bad_movies")
	if err != nil {
		t.Fatal(err)
	}

	// Change to a temporary directory.
	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err = fr.ExecCommand("qri checkout me/bad_movies"); err != nil {
		t.Fatal(err)
	}

	_ = fr.ChdirToWorkDir("bad_movies")

	// Modify the meta.json so that it fails to parse.
	if err = ioutil.WriteFile("meta.json", []byte(`{"title": "hello}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Status, check that status shows the parse error.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	// Change to a temporary directory.
	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err = fr.ExecCommand("qri checkout me/bad_body"); err != nil {
		t.Fatal(err)
	}

	_ = fr.ChdirToWorkDir("bad_body")

	// Modify the meta.json so that it fails to parse.
	if err = ioutil.WriteFile("body.json", []byte(`{"title": "hello}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Status, check that status shows the parse error.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
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

// Test that parse errors are also properly shown for structure.
func TestStatusParseErrorForStructure(t *testing.T) {
	fr := NewFSITestRunner(t, "qri_test_status_parse_error_for_structure")
	defer fr.Delete()

	// Save a dataset containing a body.json and meta component
	err := fr.ExecCommand("qri save --body=testdata/movies/body_ten.csv me/ten_movies")
	if err != nil {
		t.Fatal(err)
	}

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err = fr.ExecCommand("qri checkout me/ten_movies"); err != nil {
		t.Fatal(err)
	}

	_ = fr.ChdirToWorkDir("ten_movies")

	// Modify the meta.json so that it fails to parse.
	if err = ioutil.WriteFile("structure.json", []byte(`{"format":`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Status, check that status shows the parse error.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
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
	fr := NewFSITestRunner(t, "qri_test_status_at_version")
	defer fr.Delete()

	// First version has only a body
	err := fr.ExecCommand("qri save --body=testdata/movies/body_two.json me/status_ver")
	if err != nil {
		t.Fatal(err)
	}
	ref1 := parseRefFromSave(fr.GetCommandOutput())

	// Add a meta
	err = fr.ExecCommand("qri save --file=testdata/movies/meta_override.yaml me/status_ver")
	if err != nil {
		t.Fatal(err)
	}
	ref2 := parseRefFromSave(fr.GetCommandOutput())

	// Change the meta
	err = fr.ExecCommand("qri save --file=testdata/movies/meta_another.yaml me/status_ver")
	if err != nil {
		t.Fatal(err)
	}
	ref3 := parseRefFromSave(fr.GetCommandOutput())

	// Change the body
	err = fr.ExecCommand("qri save --body=testdata/movies/body_four.json me/status_ver")
	if err != nil {
		t.Fatal(err)
	}
	ref4 := parseRefFromSave(fr.GetCommandOutput())

	// Status for the first version of the dataset, both body and schema were added.
	if err = fr.ExecCommand(fmt.Sprintf("qri status %s", ref1)); err != nil {
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
	expect := `  structure: add
  body: add
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Status for the second version, meta added.
	if err = fr.ExecCommand(fmt.Sprintf("qri status %s", ref2)); err != nil {
		t.Fatal(err)
	}

	output = fr.GetCommandOutput()
	expect = `  meta: add
  structure: unmodified
  body: unmodified
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Status for the third version, meta modified.
	if err = fr.ExecCommand(fmt.Sprintf("qri status %s", ref3)); err != nil {
		t.Fatal(err)
	}

	output = fr.GetCommandOutput()
	expect = `  meta: modified
  structure: unmodified
  body: unmodified
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Status for the fourth version, body modified.
	if err = fr.ExecCommand(fmt.Sprintf("qri status %s", ref4)); err != nil {
		t.Fatal(err)
	}

	output = fr.GetCommandOutput()
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
	fr := NewFSITestRunner(t, "qri_test_checkout_and_restore")
	defer fr.Delete()

	// Save a dataset containing a body.csv and meta.
	err := fr.ExecCommand("qri save --body=testdata/movies/body_ten.csv --file=testdata/movies/meta_override.yaml me/ten_movies")
	if err != nil {
		t.Fatal(err)
	}

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err = fr.ExecCommand("qri checkout me/ten_movies"); err != nil {
		t.Fatal(err)
	}

	_ = fr.ChdirToWorkDir("ten_movies")

	// Modify meta.json by changing the title.
	if err = ioutil.WriteFile("meta.json", []byte(`{"title": "hello"}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Status to check that the meta is changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	// Status again, to validate that meta is no longer changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
	}

	output = fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/ten_movies"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify struture.json
	if err = ioutil.WriteFile("structure.json", []byte(`{ "format" : "csv", "schema": {"type": "array"}}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Status to check that the schema is changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
	}

	output = fr.GetCommandOutput()
	expect = `for linked dataset [test_peer/ten_movies]

  modified: structure (source: structure.json)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Restore to get the old schema back.
	if err = fr.ExecCommand("qri restore structure"); err != nil {
		t.Fatal(err)
	}

	// Status again, to validate that schema is no longer changed.
	if err = fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	_ = parseRefFromSave(fr.GetCommandOutput())

	// Add a meta
	err = fr.ExecCommand("qri save --file=testdata/movies/meta_override.yaml me/prev_ver")
	if err != nil {
		t.Fatal(err)
	}
	ref2 := parseRefFromSave(fr.GetCommandOutput())

	// Change the meta
	err = fr.ExecCommand("qri save --file=testdata/movies/meta_another.yaml me/prev_ver")
	if err != nil {
		t.Fatal(err)
	}
	_ = parseRefFromSave(fr.GetCommandOutput())

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err := fr.ExecCommand("qri checkout me/prev_ver"); err != nil {
		t.Fatal(err)
	}

	_ = fr.ChdirToWorkDir("prev_ver")

	// Verify that the status is clean
	if err = fr.ExecCommand(fmt.Sprintf("qri status")); err != nil {
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/prev_ver"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Read meta.json, contains the contents of meta_another.yaml
	metaContents, err := ioutil.ReadFile("meta.json")
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	// Read meta.json, due to restore, it has the old data from meta_override.yaml
	metaContents, err = ioutil.ReadFile("meta.json")
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	_ = parseRefFromSave(fr.GetCommandOutput())

	fr.ChdirToRoot()

	// Checkout the newly created dataset.
	if err := fr.ExecCommand("qri checkout me/del_cmp"); err != nil {
		t.Fatal(err)
	}

	workDir := fr.ChdirToWorkDir("del_cmp")

	// Modify the body.json file.
	modifyFileUsingStringReplace("body.csv", "Avatar", "The Avengers")

	// Modify meta.json by changing the title.
	if err = ioutil.WriteFile("meta.json", []byte(`{"title": "hello"}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Restore to erase the meta component.
	if err := fr.ExecCommand("qri restore meta"); err != nil {
		t.Fatal(err)
	}

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	// Restore to get erase the meta component.
	if err := fr.ExecCommand("qri restore meta"); err != nil {
		t.Fatal(err)
	}

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
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
	fr := NewFSITestRunner(t, "qri_test_render_readme")
	defer fr.Delete()

	_ = fr.CreateAndChdirToWorkDir("render_readme")

	// Init as a linked directory.
	if err := fr.ExecCommand("qri init --name render_readme --format csv"); err != nil {
		t.Fatal(err)
	}

	// Create readme.md with some text.
	if err := ioutil.WriteFile("readme.md", []byte("# hi\nhello\n"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Status, check that the working directory has added files including readme.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
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
	if err := fr.ExecCommand("qri save"); err != nil {
		t.Fatal(err)
	}

	// Status again, check that the working directory is clean.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
	}

	output = fr.GetCommandOutput()
	if diff := cmpTextLines(cleanStatusMessage("test_peer/render_readme"), output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Render the readme, check the html.
	if err := fr.ExecCommand("qri render"); err != nil {
		t.Fatal(err)
	}

	output = fr.GetCommandOutput()
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
	fr := NewFSITestRunner(t, "qri_test_init_source_body_path")
	defer fr.Delete()

	sourceFile, err := filepath.Abs("testdata/days_of_week.csv")
	if err != nil {
		panic(err)
	}

	workDir := fr.CreateAndChdirToWorkDir("init_source")

	// Init with a source body path.
	if err := fr.ExecCommand(fmt.Sprintf("qri init --name init_source --source-body-path %s", sourceFile)); err != nil {
		t.Fatal(err)
	}

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
	structureText, err := ioutil.ReadFile(filepath.Join(workDir, "structure.json"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectText, string(structureText)); diff != "" {
		t.Errorf("structure.json contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
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
	actualBody, err := ioutil.ReadFile("body.csv")
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectBody, string(actualBody)); diff != "" {
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
		t.Fatal(err)
	}

	// Directory has been created by `qri init`
	workDir := fr.ChdirToWorkDir("init_dir")

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
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
	fr := NewFSITestRunner(t, "qri_test_diff_after_change")
	defer fr.Delete()

	workDir := fr.CreateAndChdirToWorkDir("diff_change")

	// Init as a linked directory.
	if err := fr.ExecCommand("qri init --name diff_change --format csv"); err != nil {
		t.Fatal(err)
	}

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Save the new dataset.
	if err := fr.ExecCommand("qri save"); err != nil {
		t.Fatal(err)
	}

	// Modify meta.json with a title.
	if err := ioutil.WriteFile("meta.json", []byte(`{"title": "hello"}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Modify body.csv.
	if err := ioutil.WriteFile("body.csv", []byte(`lucky,number,17
four,five,321
`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Status to see changes
	if err := fr.ExecCommand("qri status"); err != nil {
		t.Fatal(err)
	}

	output := fr.GetCommandOutput()
	expect := `for linked dataset [test_peer/diff_change]

  modified: meta (source: meta.json)
  modified: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Diff to see changes
	if err := fr.ExecCommand("qri diff"); err != nil {
		t.Fatal(err)
	}

	output = fr.GetCommandOutput()
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
