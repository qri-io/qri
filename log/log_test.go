package log

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestLogState(t *testing.T) {
	logA := Log{}
	logA, _ = PutDatasetCommit(logA, "PeerA", "CommitA", "", "created dataset")
	logA, _ = PutSuggestionUpdate(logA, "PeerA", "SuggestA", "", "Hey I'm a comment")

	logB := logA.Copy()
	logB, _ = PutDatasetCommit(logB, "PeerB", "CommitB", "CommitA", "added the stuff")
	logB, _ = PutSuggestionUpdate(logB, "PeerB", "SuggestD", "", "this so cool")

	logA, _ = PutSuggestionUpdate(logA, "PeerA", "SuggestB", "", "Another Comment")
	logA, _ = PutSuggestionUpdate(logA, "PeerA", "SuggestC", "SuggestA", "updated comment")
	logA, _ = PutSuggestionDelete(logA, "PeerA", "SuggestC")

	logA = logA.Put(logB.Ops[len(logB.Ops)-1])

	logA, _ = PutDatasetCommit(logA, "PeerA", "CommitC", "CommitA", "an edit")
	logB = logB.Put(logA.Ops[len(logA.Ops)-1])

	s := logA.State()
	data, _ := json.MarshalIndent(s, "", "  ")
	fmt.Println(string(data))
}
