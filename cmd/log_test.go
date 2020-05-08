package cmd

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLogbookCommand(t *testing.T) {
	r := NewTestRunner(t, "test_peer", "qri_test_logbook")
	defer r.Delete()

	r.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")

	r.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")

	// Cannot provide dataset reference with the --raw flag
	if err := r.ExecCommand("qri logbook me/test_movies --raw"); err == nil {
		t.Error("expected using a ref and the raw flag to error")
	}

	// ProfileID of the test user
	profileID := "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"
	// AuthorID is the hash ID of the initialization block for the author
	authorID := "dis3dq3ta5donb2xoviyyfibihpsjaiqg7l6ktbuipuflxa72qqq"
	// Logbook formatted as raw json
	template := `[{"ops":[{"type":"init","model":"user","name":"test_peer","authorID":"%s","timestamp":"timeStampHere"}],"logs":[{"ops":[{"type":"init","model":"dataset","name":"test_movies","authorID":"%s","timestamp":"timeStampHere"}],"logs":[{"ops":[{"type":"init","model":"branch","name":"main","authorID":"%s","timestamp":"timeStampHere"},{"type":"init","model":"commit","ref":"/ipfs/QmXZnsLPRy9i3xFH2dzHkWG1Pkbs8AWqdhTHCYLCX76BjT","timestamp":"timeStampHere","size":224,"note":"created dataset from body_ten.csv"},{"type":"init","model":"commit","ref":"/ipfs/QmRfZNQKVR5zspwbgwqJnpdQbPyUo7bQBuSrn1qMYC31Aq","prev":"/ipfs/QmXZnsLPRy9i3xFH2dzHkWG1Pkbs8AWqdhTHCYLCX76BjT","timestamp":"timeStampHere","size":720,"note":"body changed by 70%%"}]}]}]}]`

	// Regex that replaces the timestamp with just static text
	fixTs := regexp.MustCompile(`"(timestamp|commitTime)":\s?"[0-9TZ.:+-]*?"`)

	// Verify the raw output of the logbook
	actual := r.MustExec(t, "qri logbook --raw")
	// TODO(dustmop): Make logbook's timestamp stringifier be hot-swappable to avoid this hack.
	actual = string(fixTs.ReplaceAll([]byte(actual), []byte(`"timestamp":"timeStampHere"`)))
	expect := fmt.Sprintf(template, profileID, authorID, authorID)
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}
