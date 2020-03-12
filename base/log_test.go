package base

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/qfs/localfs"
	testPeers "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

func TestDatasetLog(t *testing.T) {
	ctx := context.Background()
	mr := newTestRepo(t)
	addCitiesDataset(t, mr)
	updateCitiesDataset(t, mr, "")

	ref := repo.MustParseDatasetRef("me/not_a_dataset")
	log, err := DatasetLog(ctx, mr, ref, -1, 0, true)
	if err == nil {
		t.Errorf("expected lookup for nonexistent log to fail")
	}

	ref = repo.MustParseDatasetRef("me/cities")
	if log, err = DatasetLog(ctx, mr, ref, 1, 0, true); err != nil {
		t.Error(err.Error())
	}
	if len(log) != 1 {
		t.Errorf("log length mismatch. expected: %d, got: %d", 1, len(log))
	}

	expect := []dsref.VersionInfo{
		{
			Username:  "peer",
			Name:      "cities",
			ProfileID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
			// TODO (b5) - use constant time to make timestamp & path comparable
			CommitTitle:   "initial commit",
			CommitMessage: "created dataset",
			MetaTitle:     "this is the new title",
			BodyFormat:    "csv",
			BodySize:      155,
			BodyRows:      5,
		},
	}

	if diff := cmp.Diff(expect, log, cmpopts.IgnoreFields(dsref.VersionInfo{}, "CommitTime"), cmpopts.IgnoreFields(dsref.VersionInfo{}, "Path")); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestDatasetLogForeign(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	// Create two logbooks, mine and theirs
	myBookPath := filepath.Join(tmpdir, "my_logbook.qfs")
	theirBookPath := filepath.Join(tmpdir, "their_logbook.qfs")

	ctx := context.Background()
	mr := newTestRepo(t).(*repo.MemRepo)
	fs := localfs.NewFS()

	// Construct a logbook for another user
	theirRefStr := "them/foreign"
	otherPeerInfo := testPeers.GetTestPeerInfo(1)
	foreignBuilder := logbook.NewLogbookTempBuilder(t, otherPeerInfo.PrivKey, "them", fs, theirBookPath)
	foreignBuilder.DatasetInit(ctx, t, theirRefStr)
	ref, err := dsref.Parse(theirRefStr)
	if err != nil {
		t.Fatal(err)
	}
	// NOTE: Need to assign ProfileID because nothing is resolving the username
	ref.ProfileID = otherPeerInfo.EncodedPeerID
	foreignBuilder.Commit(ctx, t, ref, "their commit", "QmExample")
	foreignBook := foreignBuilder.Logbook()
	foreignLog, err := foreignBook.UserDatasetRef(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}

	// Construct our own logbook, and merge in the foreign oplog
	peerInfo := testPeers.GetTestPeerInfo(0)
	builder := logbook.NewLogbookTempBuilder(t, peerInfo.PrivKey, "peer", fs, myBookPath)
	builder.AddForeign(ctx, t, foreignLog)

	// Inject that log into our mem repo
	book := builder.Logbook()
	mr.SetLogbook(book)

	// Get the log, test against expectation
	r := reporef.DatasetRef{
		Peername:  ref.Username,
		ProfileID: profile.IDB58DecodeOrEmpty(ref.ProfileID),
		Name:      ref.Name,
		Path:      ref.Path,
	}
	log, err := DatasetLog(ctx, mr, r, 1, 0, true)
	if err != nil {
		t.Error(err)
	}
	if len(log) != 1 {
		t.Errorf("log length mismatch. expected: %d, got: %d", 1, len(log))
	}
	expect := []dsref.VersionInfo{
		{
			Username:    "them",
			Name:        "foreign",
			ProfileID:   "QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD",
			Foreign:     true,
			CommitTitle: "their commit",
		},
	}

	if diff := cmp.Diff(expect, log, cmpopts.IgnoreFields(dsref.VersionInfo{}, "CommitTime"), cmpopts.IgnoreFields(dsref.VersionInfo{}, "Path")); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestDatasetLogFromHistory(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	addCitiesDataset(t, r)
	head := updateCitiesDataset(t, r, "")
	expectLen := 2

	dlog, err := DatasetLogFromHistory(ctx, r, head, 0, 100, true)
	if err != nil {
		t.Error(err)
	}
	if len(dlog) != expectLen {
		t.Fatalf("log length mismatch. expected: %d, got: %d", expectLen, len(dlog))
	}
	if dlog[0].Dataset.Meta.Title != head.Dataset.Meta.Title {
		t.Errorf("expected log with loadDataset == true to populate datasets")
	}

	dlog, err = DatasetLogFromHistory(ctx, r, head, 0, 100, false)
	if err != nil {
		t.Error(err)
	}

	if len(dlog) != expectLen {
		t.Errorf("log length mismatch. expected: %d, got: %d", expectLen, len(dlog))
	}
	if dlog[0].Dataset.Meta.Title != "" {
		t.Errorf("expected log with loadDataset == false to not load a dataset. got: %v", dlog[0].Dataset)
	}
}

func TestConstructDatasetLogFromHistory(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t).(*repo.MemRepo)

	// remove the logbook
	r.RemoveLogbook()

	// create some history
	addCitiesDataset(t, r)
	ref := updateCitiesDataset(t, r, "")

	// add the logbook back
	p, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}
	book, err := logbook.NewJournal(p.PrivKey, p.Peername, r.Filesystem(), "/map/logbook")
	if err != nil {
		t.Fatal(err)
	}
	r.SetLogbook(book)

	cities := reporef.ConvertToDsref(ref)

	// confirm no history exists:
	if _, err = book.Versions(ctx, cities, 0, 100); err == nil {
		t.Errorf("expected versions for nonexistent history to fail")
	}

	// create some history
	if err := constructDatasetLogFromHistory(ctx, r, cities); err != nil {
		t.Errorf("building dataset history: %s", err)
	}

	expect := []dsref.VersionInfo{
		{
			Username:    "peer",
			CommitTitle: "initial commit",
			BodySize:    0x9b,
			ProfileID:   "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
			Name:        "cities",
			Path:        "/map/QmaTfAQNUKqtPe2EUcCELJNprRLJWswsVPHHNhiKgZoTMR",
		},
		{
			Username:    "peer",
			CommitTitle: "initial commit",
			BodySize:    0x9b,
			ProfileID:   "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
			Name:        "cities",
		},
	}

	// confirm history exists:
	got, err := book.Versions(ctx, cities, 0, 100)
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(expect, got, cmpopts.IgnoreFields(dsref.VersionInfo{}, "CommitTime"), cmpopts.IgnoreFields(dsref.VersionInfo{}, "Path")); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}
