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
)

// Test using "init" to create a new linked directory, using status to see the added files,
// then saving to create the dataset, leading to a clean status in the directory.
func TestInitStatusSave(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	r := NewTestRepoRoot(t, "qri_test_init_status_save")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	rootPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	workPath := filepath.Join(rootPath, "brand_new")
	err = os.MkdirAll(workPath, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	// Change to a temporary directory.
	os.Chdir(workPath)
	defer os.Chdir(pwd)

	// Init as a linked directory.
	cmdR := r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri init --name brand_new --format csv")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workPath)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "schema.json"}
	if diff := cmp.Diff(dirContents, expectContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory has added files.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output := r.GetOutput()
	expect := `for linked dataset [test_peer/brand_new]

  add: meta (source: meta.json)
  add: schema (source: schema.json)
  add: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Save the new dataset.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// TODO: Verify that files are in ipfs repo.

	// Status again, check that the working directory is clean.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output = r.GetOutput()
	expect = `for linked dataset [test_peer/brand_new]

working directory clean
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test that checkout, used on a simple dataset with a body.json and no meta, creates a
// working directory with a clean status.
func TestCheckoutSimpleStatus(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	r := NewTestRepoRoot(t, "qri_test_checkout_simple_status")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	rootPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	// Save a dataset containing a body.json, no meta, nothing special.
	cmdR := r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_two.json me/two_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Change to a temporary directory.
	os.Chdir(rootPath)
	defer os.Chdir(pwd)

	// Checkout the newly created dataset.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri checkout me/two_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	workPath := filepath.Join(rootPath, "two_movies")
	os.Chdir(workPath)
	defer os.Chdir(pwd)

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workPath)
	expectContents := []string{".qri-ref", "body.json", "schema.json"}
	if diff := cmp.Diff(dirContents, expectContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory is clean.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output := r.GetOutput()
	expect := `for linked dataset [test_peer/two_movies]

working directory clean
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify the body.json file.
	modifyFileUsingStringReplace("body.json", "Avatar", "The Avengers")

	// Status again, check that the body is changed.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output = r.GetOutput()
	expect = `for linked dataset [test_peer/two_movies]

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
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output = r.GetOutput()
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
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	r := NewTestRepoRoot(t, "qri_test_checkout_with_structure")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	rootPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	// Save a dataset containing a body.csv, no meta, nothing special.
	cmdR := r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_ten.csv --file=testdata/movies/meta_override.yaml me/ten_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Change to a temporary directory.
	os.Chdir(rootPath)
	defer os.Chdir(pwd)

	// Checkout the newly created dataset.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri checkout me/ten_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	workPath := filepath.Join(rootPath, "ten_movies")
	os.Chdir(workPath)
	defer os.Chdir(pwd)

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workPath)
	expectContents := []string{".qri-ref", "body.csv", "dataset.json", "meta.json", "schema.json"}
	if diff := cmp.Diff(dirContents, expectContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory is clean.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output := r.GetOutput()
	expect := `for linked dataset [test_peer/ten_movies]

working directory clean
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Modify the body.json file.
	modifyFileUsingStringReplace("body.csv", "Avatar", "The Avengers")

	// Status again, check that the body is changed.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output = r.GetOutput()
	expect = `for linked dataset [test_peer/ten_movies]

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
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output = r.GetOutput()
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
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output = r.GetOutput()
	expect = `for linked dataset [test_peer/ten_movies]

  removed:  meta
  modified: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test checkout and modifying schema, then checking status.
func TestCheckoutAndModifySchema(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	r := NewTestRepoRoot(t, "qri_test_checkout_and_modify_schema")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	rootPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	// Save a dataset containing a body.csv, no meta, nothing special.
	cmdR := r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_ten.csv me/more_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Change to a temporary directory.
	os.Chdir(rootPath)
	defer os.Chdir(pwd)

	// Checkout the newly created dataset.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri checkout me/more_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	workPath := filepath.Join(rootPath, "more_movies")
	os.Chdir(workPath)
	defer os.Chdir(pwd)

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workPath)
	expectContents := []string{".qri-ref", "body.csv", "dataset.json", "schema.json"}
	if diff := cmp.Diff(dirContents, expectContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Status, check that the working directory is clean.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output := r.GetOutput()
	expect := `for linked dataset [test_peer/more_movies]

working directory clean
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Create schema.json with a minimal schema.
	if err = ioutil.WriteFile("schema.json", []byte(`{"type": "array"}`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Status again, check that the body is changed.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output = r.GetOutput()
	// TODO(dlong): structure/dataset.json should not be marked as `modified`
	expect = `for linked dataset [test_peer/more_movies]

  modified: structure (source: dataset.json)
  modified: schema (source: schema.json)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test that status displays parse errors correctly
func TestStatusParseError(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	r := NewTestRepoRoot(t, "qri_test_status_parse_error")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	rootPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	// Save a dataset containing a body.json and meta component
	cmdR := r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_two.json --file=testdata/movies/meta_override.yaml me/bad_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Change to a temporary directory.
	os.Chdir(rootPath)
	defer os.Chdir(pwd)

	// Checkout the newly created dataset.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri checkout me/bad_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	workPath := filepath.Join(rootPath, "bad_movies")
	os.Chdir(workPath)
	defer os.Chdir(pwd)

	// Modify the meta.json so that it fails to parse.
	if err = ioutil.WriteFile("meta.json", []byte(`{"title": "hello}`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Status, check that status shows the parse error.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output := r.GetOutput()
	expect := `for linked dataset [test_peer/bad_movies]

  parse error: meta (source: meta.json)

fix these problems before saving this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test that parse errors are also properly shown for schema.
func TestStatusParseErrorForSchema(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	r := NewTestRepoRoot(t, "qri_test_status_parse_error_for_schema")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	rootPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	// Save a dataset containing a body.json and meta component
	cmdR := r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_ten.csv me/ten_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Change to a temporary directory.
	os.Chdir(rootPath)
	defer os.Chdir(pwd)

	// Checkout the newly created dataset.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri checkout me/ten_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	workPath := filepath.Join(rootPath, "ten_movies")
	os.Chdir(workPath)
	defer os.Chdir(pwd)

	// Modify the meta.json so that it fails to parse.
	if err = ioutil.WriteFile("schema.json", []byte(`{"type":`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Status, check that status shows the parse error.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri status")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output := r.GetOutput()
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
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r := NewTestRepoRoot(t, "qri_test_status_at_version")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	// Add a version with just a body
	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --body=testdata/movies/body_two.json me/status_ver")
	if err != nil {
		t.Fatalf(err.Error())
	}
	ref1 := parseRefFromSave(r.GetOutput())

	// Add a meta
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --file=testdata/movies/meta_override.yaml me/status_ver")
	if err != nil {
		t.Fatalf(err.Error())
	}
	ref2 := parseRefFromSave(r.GetOutput())

	// Change the meta
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --file=testdata/movies/meta_another.yaml me/status_ver")
	if err != nil {
		t.Fatalf(err.Error())
	}
	ref3 := parseRefFromSave(r.GetOutput())

	// Change the body
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_four.json me/status_ver")
	if err != nil {
		t.Fatalf(err.Error())
	}
	ref4 := parseRefFromSave(r.GetOutput())

	// Status for the first version of the dataset, both body and schema were added.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, fmt.Sprintf("qri status %s", ref1))
	if err != nil {
		t.Fatalf(err.Error())
	}

	output := r.GetOutput()
	expect := `  schema: add
  body: add
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Status for the second version, meta added.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, fmt.Sprintf("qri status %s", ref2))
	if err != nil {
		t.Fatalf(err.Error())
	}

	output = r.GetOutput()
	expect = `  meta: add
  schema: unmodified
  body: unmodified
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Status for the third version, meta modified.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, fmt.Sprintf("qri status %s", ref3))
	if err != nil {
		t.Fatalf(err.Error())
	}

	output = r.GetOutput()
	expect = `  meta: modified
  schema: unmodified
  body: unmodified
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Status for the fourth version, body modified.
	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, fmt.Sprintf("qri status %s", ref4))
	if err != nil {
		t.Fatalf(err.Error())
	}

	output = r.GetOutput()
	expect = `  meta: unmodified
  schema: unmodified
  body: modified
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
