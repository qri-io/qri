package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func parsePathFromRef(ref string) string {
	pos := strings.Index(ref, "@")
	if pos == -1 {
		return ref
	}
	return ref[pos+1:]
}

// Test that adding two versions, then deleting one, ends up with only the first version
func TestRemoveOneRevisionFromRepo(t *testing.T) {
	run := NewTestRunner(t, "test_peer_remote_one_rev_from_repo", "qri_test_remove_one_rev_from_repo")
	defer run.Delete()

	// Save a dataset containing a body.json, no meta, nothing special.
	output := run.MustExecCombinedOutErr(t, "qri save --body=testdata/movies/body_two.json me/remove_test")
	ref1 := parsePathFromRef(parseRefFromSave(output))
	dsPath1 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath1 {
		t.Fatalf("ref from first save should match what is in qri repo. got %q want %q", ref1, dsPath1)
	}

	// Save another version
	output = run.MustExecCombinedOutErr(t, "qri save --body=testdata/movies/body_four.json me/remove_test")
	ref2 := parsePathFromRef(parseRefFromSave(output))
	dsPath2 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath2 {
		t.Fatalf("ref from second save should match what is in qri repo. got %q want %q", ref2, dsPath2)
	}

	// Remove one version
	run.MustExec(t, "qri remove --revisions=1 me/remove_test")

	// Verify that dsref of HEAD is the same as the result of the first save command
	dsPath3 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath3 {
		t.Errorf("after delete, ref should match the first version, expected: %s\n, got: %s\n",
			ref1, dsPath3)
	}
}

// Test that adding two versions, then deleting all will end up with nothing left
func TestRemoveAllRevisionsFromRepo(t *testing.T) {
	run := NewTestRunner(t, "test_peer_remove_all_rev_", "qri_test_remove_all_rev_from_repo")
	defer run.Delete()

	// Save a dataset containing a body.json, no meta, nothing special.
	output := run.MustExecCombinedOutErr(t, "qri save --body=testdata/movies/body_two.json me/remove_test")
	ref1 := parsePathFromRef(parseRefFromSave(output))
	dsPath1 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath1 {
		t.Fatal("ref from first save should match what is in qri repo")
	}

	// Save another version
	output = run.MustExecCombinedOutErr(t, "qri save --body=testdata/movies/body_four.json me/remove_test")
	ref2 := parsePathFromRef(parseRefFromSave(output))
	dsPath2 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath2 {
		t.Fatal("ref from second save should match what is in qri repo")
	}

	// Remove one version
	run.MustExec(t, "qri remove --all me/remove_test")

	// Verify that dsref of HEAD is the same as the result of the first save command
	dsPath3 := run.GetPathForDataset(t, 0)
	if dsPath3 != "" {
		t.Errorf("after delete, dataset should not exist, got: %s\n", dsPath3)
	}
}

// Test that remove from a repo can't be used with --keep-files flag
func TestRemoveRepoCantUseKeepFiles(t *testing.T) {
	run := NewTestRunner(t, "test_peer_remove_repo_cant_use_keep_files", "qri_test_remove_repo_cant_use_keep_files")
	defer run.Delete()

	// Save a dataset containing a body.json, no meta, nothing special.
	output := run.MustExecCombinedOutErr(t, "qri save --body=testdata/movies/body_two.json me/remove_test")
	ref1 := parsePathFromRef(parseRefFromSave(output))
	dsPath1 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath1 {
		t.Fatal("ref from first save should match what is in qri repo")
	}

	// Save another version
	output = run.MustExecCombinedOutErr(t, "qri save --body=testdata/movies/body_four.json me/remove_test")
	ref2 := parsePathFromRef(parseRefFromSave(output))
	dsPath2 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath2 {
		t.Fatal("ref from second save should match what is in qri repo")
	}

	// Try to remove with the --keep-files flag should produce an error
	err := run.ExecCommand("qri remove --revisions=1 --keep-files me/remove_test")
	if err == nil {
		t.Fatal("expected error trying to remove with --keep-files, did not get an error")
	}
	expect := `dataset is not linked to filesystem, cannot use keep-files`
	if err.Error() != expect {
		t.Errorf("error mismatch, expect: %s, got: %s", expect, err.Error())
	}
}

// Test removing a revision from a linked directory
func TestRemoveOneRevisionFromWorkingDirectory(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_one_work_dir", "qri_test_remove_one_work_dir")
	defer run.Delete()

	_ = run.CreateAndChdirToWorkDir("remove_one")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name remove_one --format csv")

	// Add a meta.json.
	run.MustWriteFile(t, "meta.json", "{\"title\":\"one\"}\n")

	// Save the new dataset.
	output := run.MustExecCombinedOutErr(t, "qri save")
	ref1 := parsePathFromRef(parseRefFromSave(output))
	dsPath1 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath1 {
		t.Fatal("ref from first save should match what is in qri repo")
	}

	// Modify meta.json and body.csv.
	run.MustWriteFile(t, "meta.json", "{\"title\":\"two\"}\n")
	run.MustWriteFile(t, "body.csv", "seven,eight,9\n")

	// Save the new dataset.
	output = run.MustExecCombinedOutErr(t, "qri save")
	ref2 := parsePathFromRef(parseRefFromSave(output))
	dsPath2 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath2 {
		t.Fatal("ref from second save should match what is in qri repo")
	}

	// Remove one revision
	run.MustExec(t, "qri remove --revisions=1")

	// Verify that dsref of HEAD is the same as the result of the first save command
	dsPath3 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath3 {
		t.Errorf("after delete, ref should match the first version, expected: %s\n, got: %s\n",
			ref1, dsPath3)
	}

	// Verify the meta.json contains the original contents, not `{"title":"two"}`
	actual := run.MustReadFile(t, "meta.json")
	expect := `{
 "qri": "md:0",
 "title": "one"
}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("meta.json contents (-want +got):\n%s", diff)
	}

	// Verify the body.csv contains the original contents, not "seven,eight,9"
	actual = run.MustReadFile(t, "body.csv")
	expect = "one,two,3\nfour,five,6\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("body.csv contents (-want +got):\n%s", diff)
	}

	// Verify that status is clean
	output = run.MustExecCombinedOutErr(t, "qri status")
	expect = `for linked dataset [test_peer_remove_one_work_dir/remove_one]

working directory clean
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Verify that we can access the working directory. This would not be the case if the
	// delete operation caused the FSIPath to be moved from the dataset ref in the repo.
	actual = run.MustExecCombinedOutErr(t, "qri get")
	expect = `for linked dataset [test_peer_remove_one_work_dir/remove_one]

bodyPath: /tmp/remove_one/body.csv
meta:
  qri: md:0
  title: one
name: remove_one
peername: test_peer_remove_one_work_dir
qri: ds:0
structure:
  format: csv
  formatConfig:
    lazyQuotes: true
  qri: st:0
  schema:
    items:
      items:
      - title: field_1
        type: string
      - title: field_2
        type: string
      - title: field_3
        type: integer
      type: array
    type: array

`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("dataset result from get: (-want +got):\n%s", diff)
	}
}

// Test removing a revision which added a component will cause that component's file to be removed
func TestRemoveOneRevisionWillDeleteFilesThatWereNotThereBefore(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_one_that_wasnt_there", "qri_test_remove_one_that_wasnt_there")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("remove_one")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name remove_one --format csv")

	// Save the new dataset.
	output := run.MustExecCombinedOutErr(t, "qri save")
	ref1 := parsePathFromRef(parseRefFromSave(output))
	dsPath1 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath1 {
		t.Fatal("ref from first save should match what is in qri repo")
	}

	// Modify meta.json and body.csv.
	run.MustWriteFile(t, "meta.json", "{\"title\":\"two\"}\n")
	run.MustWriteFile(t, "body.csv", "seven,eight,9\n")

	// Save the new dataset.
	output = run.MustExecCombinedOutErr(t, "qri save")
	ref2 := parsePathFromRef(parseRefFromSave(output))
	dsPath2 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath2 {
		t.Fatal("ref from second save should match what is in qri repo")
	}

	// Remove one revision
	run.MustExec(t, "qri remove --revisions=1")

	// Verify that dsref of HEAD is the same as the result of the first save command
	dsPath3 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath3 {
		t.Errorf("after delete, ref should match the first version, expected: %s\n, got: %s\n",
			ref1, dsPath3)
	}

	// Verify that status is clean
	output = run.MustExecCombinedOutErr(t, "qri status")
	expect := `for linked dataset [test_peer_remove_one_that_wasnt_there/remove_one]

working directory clean
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}

	// Verify the directory contains the body, but not the meta
	dirContents := listDirectory(workDir)
	// TODO(dlong): meta.json is written, but is empty. Need to figure out how to determine
	// not to write it, without breaking other things.
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}
}

// Test removing a dataset with no history works
func TestRemoveNoHistory(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_no_history", "qri_test_remove_no_history")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("remove_no_history")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name remove_no_history --format csv")

	// Try to remove, but this will result in an error because working directory is not clean
	err := run.ExecCommand("qri remove --revisions=1")
	if err == nil {
		t.Fatal("expected error because working directory is not clean")
	}
	expect := `dataset not removed`
	if err.Error() != expect {
		t.Errorf("error mismatch, expect: %s, got: %s", expect, err.Error())
	}

	// Remove one revision, forced
	run.MustExec(t, "qri remove --revisions=1 -f")

	// Verify that dsref of HEAD is empty
	dsPath3 := run.GetPathForDataset(t, 0)
	if dsPath3 != "" {
		t.Errorf("after delete, ref should be empty, got: %s", dsPath3)
	}

	// Verify the directory no longer exists
	if _, err = os.Stat(workDir); !os.IsNotExist(err) {
		t.Errorf("expected \"%s\" to not exist", workDir)
	}
}

// Test removing a revision while keeping the files the same
func TestRemoveKeepFiles(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_one_keep_files", "qri_test_remove_one_keep_files")
	defer run.Delete()

	_ = run.CreateAndChdirToWorkDir("remove_one")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name remove_one --format csv")

	// Save the new dataset.
	output := run.MustExecCombinedOutErr(t, "qri save")
	ref1 := parsePathFromRef(parseRefFromSave(output))
	dsPath1 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath1 {
		t.Fatal("ref from first save should match what is in qri repo")
	}

	// Modify body.csv.
	run.MustWriteFile(t, "body.csv", "seven,eight,9\n")

	// Save the new dataset.
	output = run.MustExecCombinedOutErr(t, "qri save")
	ref2 := parsePathFromRef(parseRefFromSave(output))
	dsPath2 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath2 {
		t.Fatal("ref from second save should match what is in qri repo")
	}

	// Modify body.csv again.
	run.MustWriteFile(t, "body.csv", "ten,eleven,12\n")

	// Try to remove, but this will result in an error because working directory is not clean
	err := run.ExecCommand("qri remove --revisions=1")
	if err == nil {
		t.Fatal("expected error because working directory is not clean")
	}
	expect := `dataset not removed`
	if err.Error() != expect {
		t.Errorf("error mismatch, expect: %s, got: %s", expect, err.Error())
	}

	// Verify that dsref of HEAD is still the result of the second save
	dsPath3 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath3 {
		t.Errorf("no commits should have been removed, expected: %s\n, got: %s\n",
			ref2, dsPath3)
	}

	// Remove is possible using --keep-files
	run.MustExec(t, "qri remove --revisions=1 --keep-files")

	// Verify that dsref is now the result of the first save because one commit was removed
	dsPath4 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath4 {
		t.Errorf("no commits should have been removed, expected: %s\n, got: %s\n",
			ref1, dsPath4)
	}

	// Verify the body.csv contains the newest version and was not removed
	actual := run.MustReadFile(t, "body.csv")
	expect = "ten,eleven,12\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("body.csv contents (-want +got):\n%s", diff)
	}

	// Verify that status is dirty because we kept the files
	output = run.MustExecCombinedOutErr(t, "qri status")
	expect = `for linked dataset [test_peer_remove_one_keep_files/remove_one]

  modified: body (source: body.csv)

run ` + "`qri save`" + ` to commit this dataset
`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri status (-want +got):\n%s", diff)
	}
}

// Test removing all versions from a working directory
func TestRemoveAllVersionsWorkingDirectory(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_all_work_dir", "qri_test_remove_all_work_dir")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("remove_all")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name remove_all --format csv")

	// Save the new dataset.
	output := run.MustExecCombinedOutErr(t, "qri save")
	ref1 := parsePathFromRef(parseRefFromSave(output))
	dsPath1 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath1 {
		t.Fatal("ref from first save should match what is in qri repo")
	}

	// Modify body.csv.
	run.MustWriteFile(t, "body.csv", "seven,eight,9\n")

	// Save the new dataset.
	output = run.MustExecCombinedOutErr(t, "qri save")
	ref2 := parsePathFromRef(parseRefFromSave(output))
	dsPath2 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath2 {
		t.Fatal("ref from second save should match what is in qri repo")
	}

	// Remove all versions
	run.MustExec(t, "qri remove --all")

	// Verify that dsref of HEAD is empty
	dsPath3 := run.GetPathForDataset(t, 0)
	if dsPath3 != "" {
		t.Errorf("after delete, ref should be empty, got: %s", dsPath3)
	}

	// Verify the directory no longer exists
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Errorf("expected \"%s\" to not exist", workDir)
	}
}

// Test removing all versions from a working directory with low value files
func TestRemoveAllVersionsWorkingDirectoryLowValueFiles(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_all_work_low_value_files", "qri_test_remove_all_work_dir_low_value_files")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("remove_all")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name remove_all --format csv")

	// Save the new dataset.
	output := run.MustExecCombinedOutErr(t, "qri save")
	ref1 := parsePathFromRef(parseRefFromSave(output))
	dsPath1 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath1 {
		t.Fatal("ref from first save should match what is in qri repo")
	}

	// Modify body.csv.
	run.MustWriteFile(t, "body.csv", "seven,eight,9\n")

	// Save the new dataset.
	output = run.MustExecCombinedOutErr(t, "qri save")
	ref2 := parsePathFromRef(parseRefFromSave(output))
	dsPath2 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath2 {
		t.Fatal("ref from second save should match what is in qri repo")
	}

	lowValueFiles := []string{
		// generic files
		".test.swp", // Swap file for vim state

		// macOS specific files
		".DS_Store",    // Stores custom folder attributes
		".AppleDouble", // Stores additional file resources
		".LSOverride",  // Contains the absolute path to the app to be used
		"Icon\r",       // Custom Finder icon: http://superuser.com/questions/298785/icon-file-on-os-x-desktop
		"._test",       // Thumbnail
		".Trashes",     // File that might appear on external disk
		"__MACOSX",     // Resource fork

		// Windows specific files
		"Thumbs.db",   // Image file cache
		"ehthumbs.db", // Folder config file
		"Desktop.ini", // Stores custom folder attributes

	}

	// Add low value files
	for _, file := range lowValueFiles {
		run.MustWriteFile(t, file, "\n")
	}

	// Remove all versions
	run.MustExec(t, "qri remove --all")

	// Verify that dsref of HEAD is empty
	dsPath3 := run.GetPathForDataset(t, 0)
	if dsPath3 != "" {
		t.Errorf("after delete, ref should be empty, got: %s", dsPath3)
	}

	// Verify the directory still exists
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		t.Errorf("expected \"%s\" to still exist", workDir)
	}
}

// Test removing all versions from a working directory with --force due to low value files
func TestRemoveAllForceVersionsWorkingDirectoryLowValueFiles(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_all_work_dir_force", "qri_test_remove_all_work_dir_force")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("remove_all")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name remove_all --format csv")

	// Save the new dataset.
	output := run.MustExecCombinedOutErr(t, "qri save")
	ref1 := parsePathFromRef(parseRefFromSave(output))
	dsPath1 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath1 {
		t.Fatal("ref from first save should match what is in qri repo")
	}

	// Modify body.csv.
	run.MustWriteFile(t, "body.csv", "seven,eight,9\n")

	// Save the new dataset.
	output = run.MustExecCombinedOutErr(t, "qri save")
	ref2 := parsePathFromRef(parseRefFromSave(output))
	dsPath2 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath2 {
		t.Fatal("ref from second save should match what is in qri repo")
	}

	lowValueFiles := []string{
		// generic files
		".test.swp", // Swap file for vim state

		// macOS specific files
		".DS_Store",    // Stores custom folder attributes
		".AppleDouble", // Stores additional file resources
		".LSOverride",  // Contains the absolute path to the app to be used
		"Icon\r",       // Custom Finder icon: http://superuser.com/questions/298785/icon-file-on-os-x-desktop
		"._test",       // Thumbnail
		".Trashes",     // File that might appear on external disk
		"__MACOSX",     // Resource fork

		// Windows specific files
		"Thumbs.db",   // Image file cache
		"ehthumbs.db", // Folder config file
		"Desktop.ini", // Stores custom folder attributes

	}

	// Add low value files
	for _, file := range lowValueFiles {
		run.MustWriteFile(t, file, "\n")
	}

	// Remove all versions
	run.MustExec(t, "qri remove --all --force")

	// Verify that dsref of HEAD is empty
	dsPath3 := run.GetPathForDataset(t, 0)
	if dsPath3 != "" {
		t.Errorf("after delete, ref should be empty, got: %s", dsPath3)
	}

	// Verify the directory still exists
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Errorf("expected \"%s\" to not exist", workDir)
	}
}

// Test removing all versions while keeping files
func TestRemoveAllAndKeepFiles(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_all_keep_files", "qri_test_remove_all_keep_files")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("remove_all")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name remove_all --format csv")

	// Save the new dataset.
	output := run.MustExecCombinedOutErr(t, "qri save")
	ref1 := parsePathFromRef(parseRefFromSave(output))
	dsPath1 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath1 {
		t.Fatal("ref from first save should match what is in qri repo")
	}

	// Modify body.csv.
	run.MustWriteFile(t, "body.csv", "seven,eight,9\n")

	// Save the new dataset.
	output = run.MustExecCombinedOutErr(t, "qri save")
	ref2 := parsePathFromRef(parseRefFromSave(output))
	dsPath2 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath2 {
		t.Fatal("ref from second save should match what is in qri repo")
	}

	// Remove all but --keep-files
	run.MustExec(t, "qri remove --revisions=all --keep-files")

	// Verify that dsref of HEAD is empty
	dsPath3 := run.GetPathForDataset(t, 0)
	if dsPath3 != "" {
		t.Errorf("after delete, ref should be empty, got: %s", dsPath3)
	}

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{"body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}
}

// Test removing all versions and files
func TestRemoveAllForce(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_all_force", "qri_test_remove_all_force")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("remove_all")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name remove_all --format csv")

	// Save the new dataset.
	output := run.MustExecCombinedOutErr(t, "qri save")
	ref1 := parsePathFromRef(parseRefFromSave(output))
	dsPath1 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath1 {
		t.Fatal("ref from first save should match what is in qri repo")
	}

	// Modify body.csv.
	run.MustWriteFile(t, "body.csv", "seven,eight,9\n")

	// Save the new dataset.
	output = run.MustExecCombinedOutErr(t, "qri save")
	ref2 := parsePathFromRef(parseRefFromSave(output))
	dsPath2 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath2 {
		t.Fatal("ref from second save should match what is in qri repo")
	}

	// Add low value files
	run.MustWriteFile(t, ".DS_Store", "\n")

	// Remove all with force
	run.MustExec(t, "qri remove --all --force")

	// Verify that dsref of HEAD is empty
	dsPath3 := run.GetPathForDataset(t, 0)
	if dsPath3 != "" {
		t.Errorf("after delete, ref should be empty, got: '%s'", dsPath3)
	}

	// Verify the directory longer still exists
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Errorf("expected \"%s\" to not exist", workDir)
	}
}

// Test removing all versions and files, should fail to remove the directory
// if other files are present (not including low value files)
func TestRemoveAllForceShouldFailIfDirty(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_all_force_fail_if_dirty", "qri_test_remove_all_force_fail_if_dirty")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("remove_all")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name remove_all --format csv")

	// Save the new dataset.
	output := run.MustExecCombinedOutErr(t, "qri save")
	ref1 := parsePathFromRef(parseRefFromSave(output))
	dsPath1 := run.GetPathForDataset(t, 0)
	if ref1 != dsPath1 {
		t.Fatal("ref from first save should match what is in qri repo")
	}

	// Modify body.csv.
	run.MustWriteFile(t, "body.csv", "seven,eight,9\n")

	// Save the new dataset.
	output = run.MustExecCombinedOutErr(t, "qri save")
	ref2 := parsePathFromRef(parseRefFromSave(output))
	dsPath2 := run.GetPathForDataset(t, 0)
	if ref2 != dsPath2 {
		t.Fatal("ref from second save should match what is in qri repo")
	}

	// Add low value files
	run.MustWriteFile(t, ".DS_Store", "\n")
	// Add other files
	run.MustWriteFile(t, "test.sh", "echo test\n")

	// Remove all with force
	run.MustExec(t, "qri remove --all --force")

	// Verify that dsref of HEAD is empty
	dsPath3 := run.GetPathForDataset(t, 0)
	if dsPath3 != "" {
		t.Errorf("after delete, ref should be empty, got: %s", dsPath3)
	}

	// Verify the directory no longer exists
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		t.Errorf("expected \"%s\" to still exist", workDir)
	}
	// Verify other files still exist
	actual := run.MustReadFile(t, "test.sh")
	expect := "echo test\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("test.sh contents (-want +got):\n%s", diff)
	}
}

// Test removing a linked dataset after the working directory has already been deleted.
func TestRemoveIfWorkingDirectoryIsNotFound(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_no_wd", "qri_test_remove_no_wd")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("remove_no_wd")

	// Init as a linked directory
	run.MustExec(t, "qri init --name remove_no_wd --format csv")

	// Save the new dataset
	run.MustExec(t, "qri save")

	// Go up one directory
	parentDir := filepath.Dir(workDir)
	os.Chdir(parentDir)

	// Remove the working directory
	err := os.RemoveAll(workDir)
	if err != nil {
		t.Fatal(err)
	}

	// Remove all should still work, even though the working directory is gone.
	if err = run.ExecCommand("qri remove --revisions=all me/remove_no_wd"); err != nil {
		t.Error(err)
	}
}

// Test that a dataset can be removed even if the logbook is missing
func TestRemoveEvenIfLogbookGone(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_no_logbook", "qri_test_remove_no_logbook")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("remove_no_logbook")

	// Init as a linked directory
	run.MustExec(t, "qri init --name remove_no_logbook --format csv")

	// Save the new dataset
	run.MustExec(t, "qri save")

	// Go up one directory
	parentDir := filepath.Dir(workDir)
	os.Chdir(parentDir)

	// Remove the logbook
	logbookFile := filepath.Join(run.RepoRoot.RootPath, "qri/logbook.qfb")
	if _, err := os.Stat(logbookFile); os.IsNotExist(err) {
		t.Fatal("logbook does not exist")
	}
	err := os.Remove(logbookFile)
	if err != nil {
		t.Fatal(err)
	}

	// Remove all should still work, even though the logbook is gone.
	if err := run.ExecCommand("qri remove --revisions=all me/remove_no_logbook"); err != nil {
		t.Error(err)
	}
}

// Test that an added dataset can be removed, which removes it from the logbook
func TestRemoveEvenIfForeignDataset(t *testing.T) {
	run := NewTestRunnerWithMockRemoteClient(t, "test_peer_remove_foreign", "remove_foreign")
	defer run.Delete()

	// Regex that replaces the timestamp with just static text
	fixTs := regexp.MustCompile(`"(timestamp|commitTime)":\s?"[0-9TZ.:+-]*?"`)

	output := run.MustExec(t, "qri logbook --raw")
	expectEmpty := `[{"ops":[{"type":"init","model":"user","name":"test_peer_remove_foreign","authorID":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B","timestamp":"ts"}]}]`
	actual := string(fixTs.ReplaceAll([]byte(output), []byte(`"timestamp":"ts"`)))
	if diff := cmp.Diff(expectEmpty, actual); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Save a foreign dataset
	run.MustExec(t, "qri add other_peer/their_dataset")

	output = run.MustExec(t, "qri logbook --raw")
	expectHasForiegn := `[{"ops":[{"type":"init","model":"user","name":"test_peer_remove_foreign","authorID":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B","timestamp":"ts"}]},{"ops":[{"type":"init","model":"user","name":"other_peer","authorID":"QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD","timestamp":"ts"}],"logs":[{"ops":[{"type":"init","model":"dataset","name":"their_dataset","authorID":"sizb4wwisfvzr7vkduml5e7ivp6igi6eykqnhfyy5po3wtq5r7sa","timestamp":"ts"}],"logs":[{"ops":[{"type":"init","model":"branch","name":"main","authorID":"sizb4wwisfvzr7vkduml5e7ivp6igi6eykqnhfyy5po3wtq5r7sa","timestamp":"ts"},{"type":"init","model":"commit","ref":"QmExample","timestamp":"ts","note":"their commit"}]}]}]}]`
	actual = string(fixTs.ReplaceAll([]byte(output), []byte(`"timestamp":"ts"`)))
	if diff := cmp.Diff(expectHasForiegn, actual); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Remove all should still work, even though the dataset is foreign
	if err := run.ExecCommand("qri remove --revisions=all other_peer/their_dataset"); err != nil {
		t.Error(err)
	}

	output = run.MustExec(t, "qri logbook --raw")
	// Log is removed for the database, but author init still remains
	expectEmptyAuthor := `[{"ops":[{"type":"init","model":"user","name":"test_peer_remove_foreign","authorID":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B","timestamp":"ts"}]},{"ops":[{"type":"init","model":"user","name":"other_peer","authorID":"QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD","timestamp":"ts"}]}]`
	actual = string(fixTs.ReplaceAll([]byte(output), []byte(`"timestamp":"ts"`)))
	if diff := cmp.Diff(expectEmptyAuthor, actual); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}

// Test that an added dataset can be removed even if the logbook is missing
func TestRemoveEvenIfForeignDatasetWithNoOplog(t *testing.T) {
	run := NewTestRunnerWithMockRemoteClient(t, "test_peer_no_oplog", "remove_no_oplog")
	defer run.Delete()

	// Save a foreign dataset
	run.MustExec(t, "qri add other_peer/their_dataset")

	// Remove the logbook
	logbookFile := filepath.Join(run.RepoRoot.RootPath, "qri/logbook.qfb")
	if _, err := os.Stat(logbookFile); os.IsNotExist(err) {
		t.Fatal("logbook does not exist")
	}
	err := os.Remove(logbookFile)
	if err != nil {
		t.Fatal(err)
	}

	// Remove all should still work, even though the dataset is foreign with no logbook
	if err := run.ExecCommand("qri remove --revisions=all other_peer/their_dataset"); err != nil {
		t.Error(err)
	}
}

// Test that remove can cleanup datasets in an inconsistent state
func TestRemoveWorksAfterDeletingWorkingDirectoryFromInit(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_remove_rm_wd_from_init", "qri_test_remove_rm_wd_from_init")
	defer run.Delete()

	sourceFile, err := filepath.Abs("testdata/movies/body_ten.csv")
	if err != nil {
		panic(err)
	}

	workDir := run.CreateAndChdirToWorkDir("remove_rm_wd")

	// Init as a linked directory.
	run.MustExec(t, fmt.Sprintf("qri init --name remove_rm_wd --source-body-path %s", sourceFile))

	// Go up one directory
	parentDir := filepath.Dir(workDir)
	os.Chdir(parentDir)

	// Remove the working directory
	if err = os.RemoveAll(workDir); err != nil {
		t.Fatal(err)
	}

	// Running a command will hit the "EnsureRef" path, which removes the reference from the repo.
	run.ExecCommand("qri list")

	// Reference is no longer in the refstore, but logbook did not have a delete operation written.
	// Using --force flag will put this into a consistent state.
	output := run.MustExec(t, "qri remove --all me/remove_rm_wd -f")
	if !strings.Contains(output, "removed remains of dataset from logbook") {
		t.Error("expected to clean up remains of dataset from logbook")
	}

	workDir = run.CreateAndChdirToWorkDir("remove_rm_wd")

	// Init as a linked directory.
	err = run.ExecCommand(fmt.Sprintf("qri init --name remove_rm_wd --source-body-path %s", sourceFile))
	if err != nil {
		t.Errorf("should be able to init dataset with the removed name, got %q", err)
	}
}
