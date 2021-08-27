package cmd

import (
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

// Test that a dataset can be removed even if the logbook is missing
func TestRemoveEvenIfLogbookGone(t *testing.T) {
	run := NewTestRunner(t, "test_peer_remove_no_logbook", "qri_test_remove_no_logbook")
	defer run.Delete()

	// Save the new dataset
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv me/movies")

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
	if err := run.ExecCommand("qri remove --revisions=all me/movies"); err != nil {
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
	expectHasForiegn := `[{"ops":[{"type":"init","model":"user","name":"test_peer_remove_foreign","authorID":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B","timestamp":"ts"}]},{"ops":[{"type":"init","model":"user","name":"other_peer","authorID":"QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD","timestamp":"ts"}],"logs":[{"ops":[{"type":"init","model":"dataset","name":"their_dataset","authorID":"xstfcrqf26suws6dnjih4ugvmfk6w5o7e6b7rmflt7aso6htyufa","timestamp":"ts"}],"logs":[{"ops":[{"type":"init","model":"branch","name":"main","authorID":"xstfcrqf26suws6dnjih4ugvmfk6w5o7e6b7rmflt7aso6htyufa","timestamp":"ts"},{"type":"init","model":"commit","timestamp":"ts","size":2,"note":"created dataset"}]}]}]}]`
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
