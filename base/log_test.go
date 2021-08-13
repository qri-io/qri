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
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
)

func TestDatasetLog(t *testing.T) {
	ctx := context.Background()
	mr := newTestRepo(t)
	addCitiesDataset(t, mr)
	cities := updateCitiesDataset(t, mr, "")

	ref := dsref.MustParse("me/not_a_dataset")
	log, err := DatasetLog(ctx, mr, ref, -1, 0, true)
	if err == nil {
		t.Errorf("expected lookup for nonexistent log to fail")
	}

	if log, err = DatasetLog(ctx, mr, cities, 1, 0, true); err != nil {
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
			MetaTitle:     "this is the new title",
			BodyFormat:    "csv",
			BodySize:      155,
			BodyRows:      5,
			CommitTitle:   "initial commit",
			CommitMessage: "created dataset",
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
	fs, err := localfs.NewFS(nil)
	if err != nil {
		t.Fatal(err)
	}

	// Construct a logbook for another user
	theirRefStr := "them/foreign"
	themKeyData := testkeys.GetKeyData(1)
	them, err := profile.NewSparsePKProfile("them", themKeyData.PrivKey)
	if err != nil {
		t.Fatal(err)
	}
	foreignBuilder := logbook.NewLogbookTempBuilder(t, them, fs, theirBookPath)
	ref, err := dsref.Parse(theirRefStr)
	if err != nil {
		t.Fatal(err)
	}
	initID := foreignBuilder.DatasetInit(ctx, t, ref.Name)
	// NOTE: Need to assign ProfileID because nothing is resolving the username
	ref.ProfileID = themKeyData.EncodedPeerID
	ref.Path = "/mem/QmExample"
	foreignBuilder.Commit(ctx, t, initID, "their commit", ref.Path)
	foreignBook := foreignBuilder.Logbook()
	foreignLog, err := foreignBook.UserDatasetBranchesLog(ctx, initID)
	if err != nil {
		t.Fatal(err)
	}

	// Construct our own logbook, and merge in the foreign oplog
	ourKey := testkeys.GetKeyData(0).PrivKey
	us, err := profile.NewSparsePKProfile("us", ourKey)
	if err != nil {
		t.Fatal(err)
	}
	builder := logbook.NewLogbookTempBuilder(t, us, fs, myBookPath)
	builder.AddForeign(ctx, t, foreignLog)

	// Inject that log into our mem repo
	book := builder.Logbook()
	mr.SetLogbook(book)

	log, err := DatasetLog(ctx, mr, ref, 1, 0, true)
	if err != nil {
		t.Error(err)
	}
	if len(log) != 1 {
		t.Errorf("log length mismatch. expected: %d, got: %d", 1, len(log))
	}

	// Foreign log so not counting on the commit message
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

func TestDatasetLogForeignTimeout(t *testing.T) {
	ctx := context.Background()
	mr := newTestRepo(t).(*repo.MemRepo)

	// Test peer
	username := "test_peer_dataset_log_foreign_timeout"
	themKeyData := testkeys.GetKeyData(1)
	ref := dsref.Ref{
		Username: username,
		// TODO(b5): this peerID should be constructed from key.ID
		ProfileID: themKeyData.PeerID.String(),
		Name:      "foreign_ds",
		Path:      "/mem/notLocalPath",
	}

	vi := dsref.NewVersionInfoFromRef(ref)
	// Add a reference to the repo which uses a path not in our filestore
	err := repo.PutVersionInfoShim(ctx, mr, &vi)
	if err != nil {
		t.Fatal(err)
	}

	// Get a dataset log, which should timeout with an error
	_, err = DatasetLog(ctx, mr, ref, -1, 0, true)
	if err == nil {
		t.Fatal("expected lookup for foreign log to fail")
	}
	expectErr := `datasetLog: timeout`
	if err.Error() != expectErr {
		t.Errorf("error mismatch, expected %s, got %s", expectErr, err)
	}
}

func TestStoredHistoricalDatasets(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	addCitiesDataset(t, r)
	head := updateCitiesDataset(t, r, "")
	expectLen := 2

	citiesDs, err := ReadDataset(ctx, r, head.Path)
	if err != nil {
		t.Fatal(err)
	}

	datasets, err := StoredHistoricalDatasets(ctx, r, head.Path, 0, 100, true)
	if err != nil {
		t.Error(err)
	}
	if len(datasets) != expectLen {
		t.Fatalf("log length mismatch. expected: %d, got: %d", expectLen, len(datasets))
	}
	if datasets[0].Meta.Title != citiesDs.Meta.Title {
		t.Errorf("expected log with loadDataset == true to populate datasets")
	}

	datasets, err = StoredHistoricalDatasets(ctx, r, head.Path, 0, 100, false)
	if err != nil {
		t.Error(err)
	}

	if len(datasets) != expectLen {
		t.Errorf("log length mismatch. expected: %d, got: %d", expectLen, len(datasets))
	}
	if datasets[0].Meta.Title != "" {
		t.Errorf("expected log with loadDataset == false to not load a dataset. got: %v", datasets[0])
	}
}

func TestConstructDatasetLogFromHistory(t *testing.T) {
	ctx := context.Background()
	mr := newTestRepo(t).(*repo.MemRepo)

	// remove the logbook
	mr.RemoveLogbook()

	// create some history
	addCitiesDataset(t, mr)
	ref := updateCitiesDataset(t, mr, "")

	// add the logbook back
	p := mr.Profiles().Owner(ctx)
	book, err := logbook.NewJournal(*p, event.NilBus, mr.Filesystem(), "/mem/logbook.qfb")
	if err != nil {
		t.Fatal(err)
	}
	mr.SetLogbook(book)

	// confirm no history exists:
	if _, err = book.Items(ctx, ref, 0, 100); err == nil {
		t.Errorf("expected versions for nonexistent history to fail")
	}

	// create some history
	if err := constructDatasetLogFromHistory(ctx, mr, ref); err != nil {
		t.Errorf("building dataset history: %s", err)
	}
	expect := []dsref.VersionInfo{
		{
			Username:      "peer",
			BodySize:      0x9b,
			ProfileID:     "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
			Name:          "cities",
			CommitTitle:   "initial commit",
			CommitMessage: "created dataset",
		},
		{
			Username:      "peer",
			BodySize:      0x9b,
			ProfileID:     "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
			Name:          "cities",
			Path:          "/map/QmaTfAQNUKqtPe2EUcCELJNprRLJWswsVPHHNhiKgZoTMR",
			CommitTitle:   "initial commit",
			CommitMessage: "created dataset",
		},
	}

	// confirm history exists:
	log, err := DatasetLog(ctx, mr, ref, 100, 0, true)
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(expect, log, cmpopts.IgnoreFields(dsref.VersionInfo{}, "CommitTime"), cmpopts.IgnoreFields(dsref.VersionInfo{}, "Path")); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}
