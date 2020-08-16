package cmd

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLogbookCommand(t *testing.T) {
	r := NewTestRunner(t, "test_peer_logbook", "qri_test_logbook")
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
	authorID := "74iwd7hnx5u47nfnu73auj77hycw5ivdjswdrfyafh4cr3ylmwnq"
	// Logbook formatted as raw json
	template := `[{"ops":[{"type":"init","model":"user","name":"test_peer_logbook","authorID":"%s","timestamp":"timeStampHere"}],"logs":[{"ops":[{"type":"init","model":"dataset","name":"test_movies","authorID":"%s","timestamp":"timeStampHere"}],"logs":[{"ops":[{"type":"init","model":"branch","name":"main","authorID":"%s","timestamp":"timeStampHere"},{"type":"init","model":"commit","ref":"/ipfs/QmNX9ZKXtdskpYSQ5spd1qvqB2CPoWfJbdAcWoFndintrF","timestamp":"timeStampHere","size":224,"note":"created dataset from body_ten.csv"},{"type":"init","model":"commit","ref":"/ipfs/QmXmnH1tFKyG493wsFiisZ14N4cjrymZZ6pqK3Vr9vWS2p","prev":"/ipfs/QmNX9ZKXtdskpYSQ5spd1qvqB2CPoWfJbdAcWoFndintrF","timestamp":"timeStampHere","size":720,"note":"body changed by 70%%"}]}]}]}]`

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
