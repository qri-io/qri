package dscache

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	testPeers "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// Test the buildDscacheFlatbuffer function, which converts plain-old-data structures into dscache
func TestBuildDscacheFlatbuffer(t *testing.T) {
	pid0 := profile.IDFromPeerID(testPeers.GetTestPeerInfo(0).PeerID)
	pid1 := profile.IDFromPeerID(testPeers.GetTestPeerInfo(1).PeerID)
	pid2 := profile.IDFromPeerID(testPeers.GetTestPeerInfo(2).PeerID)

	userList := []userProfilePair{
		userProfilePair{
			Username:  "test_zero",
			ProfileID: pid0.String(),
		},
		userProfilePair{
			Username:  "test_one",
			ProfileID: pid1.String(),
		},
		userProfilePair{
			Username:  "test_two",
			ProfileID: pid2.String(),
		},
	}
	entryInfoList := []*entryInfo{
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "ds_init_id_0000",
				ProfileID: pid1.String(),
				Name:      "my_ds",
				Path:      "/ipfs/QmExampleFirst",
			},
		},
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "ds_init_id_0001",
				ProfileID: pid1.String(),
				Name:      "another_ds",
				Path:      "/ipfs/QmExampleSecond",
			},
		},
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "ds_init_id_0002",
				ProfileID: pid1.String(),
				Name:      "checked_out_ds",
				Path:      "/ipfs/QmExampleThird",
				FSIPath:   "/path/to/workspace/checked_out_ds",
			},
		},
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "ds_init_id_0003",
				ProfileID: pid2.String(),
				Name:      "foreign_ds",
				Path:      "/ipfs/QmExampleFourth",
			},
		},
	}
	dscache := buildDscacheFlatbuffer(userList, entryInfoList)
	actual := dscache.VerboseString(false)

	expect := `Dscache:
 Dscache.Users:
  0) user=test_zero profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  1) user=test_one profileID=QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
  2) user=test_two profileID=QmPeFTNHcZDr3ZFEfFfepxS5PqHAmfBRGQNPJ389Cwh1as
 Dscache.Refs:
  0) initID        = ds_init_id_0000
     profileID     = QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
     topIndex      = 0
     cursorIndex   = 0
     prettyName    = my_ds
     commitTime    = -62135596800
     headRef       = /ipfs/QmExampleFirst
  1) initID        = ds_init_id_0001
     profileID     = QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
     topIndex      = 0
     cursorIndex   = 0
     prettyName    = another_ds
     commitTime    = -62135596800
     headRef       = /ipfs/QmExampleSecond
  2) initID        = ds_init_id_0002
     profileID     = QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
     topIndex      = 0
     cursorIndex   = 0
     prettyName    = checked_out_ds
     commitTime    = -62135596800
     headRef       = /ipfs/QmExampleThird
     fsiPath       = /path/to/workspace/checked_out_ds
  3) initID        = ds_init_id_0003
     profileID     = QmPeFTNHcZDr3ZFEfFfepxS5PqHAmfBRGQNPJ389Cwh1as
     topIndex      = 0
     cursorIndex   = 0
     prettyName    = foreign_ds
     commitTime    = -62135596800
     headRef       = /ipfs/QmExampleFourth
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("builddscacheFlatbuffer (-want +got):\n%s", diff)
	}
}

type DscacheTestRunner struct {
	prevTsFunc func() int64
	minute     int
	teardown   func()
}

func NewDscacheTestRunner() *DscacheTestRunner {
	run := DscacheTestRunner{minute: 0}
	run.prevTsFunc = logbook.NewTimestamp
	logbook.NewTimestamp = func() int64 {
		run.minute++
		return time.Date(2000, time.January, 1, 0, run.minute, 0, 0, time.UTC).UnixNano()
	}
	return &run
}

func (run *DscacheTestRunner) Delete() {
	logbook.NewTimestamp = run.prevTsFunc
	if run.teardown != nil {
		run.teardown()
	}
}

// MustPutDatasetFileAtKey puts a dataset into the storage at the given key
func (run *DscacheTestRunner) MustPutDatasetFileAtKey(t *testing.T, store *cafs.MapStore, key, content string) {
	ctx := context.Background()
	err := store.PutFileAtKey(ctx, key, qfs.NewMemfileBytes("dataset.json", []byte(content)))
	if err != nil {
		t.Fatal(err)
	}
}

// Test the convertLogbookAndRefs function, which turns a logbook and dsrefs into a list of entryInfo
func TestConvertLogbookAndRefsBasic(t *testing.T) {
	run := NewDscacheTestRunner()
	defer run.Delete()

	ctx := context.Background()

	peerInfo := testPeers.GetTestPeerInfo(0)
	book := makeFakeLogbook(ctx, t, "test_user", peerInfo.PrivKey)

	dsrefs := []reporef.DatasetRef{
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.IDFromPeerID(peerInfo.PeerID),
			Name:      "first_new_name",
			Path:      "QmHashOfVersion2",
			FSIPath:   "/path/to/first_workspace",
		},
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.IDFromPeerID(peerInfo.PeerID),
			Name:      "second_name",
			Path:      "QmHashOfVersion6",
			FSIPath:   "/path/to/second_workspace",
		},
	}

	entryInfoList, err := convertLogbookAndRefs(ctx, book, dsrefs)
	if err != nil {
		t.Fatal(err)
	}

	expect := []*entryInfo{
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "htkkr2g4st3atjmxhkar3kjpv6x3xgls7sdkh4rm424v45tqpt6q",
				ProfileID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Name:      "first_new_name",
				Path:      "QmHashOfVersion2",
				FSIPath:   "/path/to/first_workspace",
			},
			TopIndex:    2,
			CursorIndex: 2,
		},
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "7n6dyt5aabo6j4fl2dbwwymoznsnd255egn6rb5cwchwetsoowzq",
				ProfileID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Name:      "second_name",
				Path:      "QmHashOfVersion6",
				FSIPath:   "/path/to/second_workspace",
			},
			TopIndex:    3,
			CursorIndex: 3,
		},
	}
	if diff := cmp.Diff(expect, entryInfoList); diff != "" {
		t.Errorf("convertLogbookAndRefs (-want +got):\n%s", diff)
	}
}

// Test that convertLogbookAndRefs still works if there's refs in logbook.
func TestConvertLogbookAndRefsMissingDsref(t *testing.T) {
	run := NewDscacheTestRunner()
	defer run.Delete()

	ctx := context.Background()

	peerInfo := testPeers.GetTestPeerInfo(0)
	book := makeFakeLogbook(ctx, t, "test_user", peerInfo.PrivKey)

	// This is missing the second dsref
	dsrefs := []reporef.DatasetRef{
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.IDFromPeerID(peerInfo.PeerID),
			Name:      "first_new_name",
			Path:      "QmHashOfVersion2",
			FSIPath:   "/path/to/first_workspace",
		},
	}

	entryInfoList, err := convertLogbookAndRefs(ctx, book, dsrefs)
	if err != nil {
		t.Fatal(err)
	}

	expect := []*entryInfo{
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "htkkr2g4st3atjmxhkar3kjpv6x3xgls7sdkh4rm424v45tqpt6q",
				ProfileID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Name:      "first_new_name",
				Path:      "QmHashOfVersion2",
				FSIPath:   "/path/to/first_workspace",
			},
			TopIndex:    2,
			CursorIndex: 2,
		},
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "7n6dyt5aabo6j4fl2dbwwymoznsnd255egn6rb5cwchwetsoowzq",
				ProfileID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Name:      "second_name",
				Path:      "QmHashOfVersion6",
			},
			TopIndex:    3,
			CursorIndex: 3,
		},
	}
	if diff := cmp.Diff(expect, entryInfoList); diff != "" {
		t.Errorf("convertLogbookAndRefs (-want +got):\n%s", diff)
	}
}

// Test a logbook missing refs from dsref will add them when making the dscache
// For now, this should only happen for users that were modifying their logbook or who created
// datasets before logbook existed. In the future, it shouldn't be necessary to explicitly
// handle this case. Eventually dsrefs will go away and only dscache will be used in this
// manner.
func TestConvertLogbookAndRefsMissingFromLogbook(t *testing.T) {
	run := NewDscacheTestRunner()
	defer run.Delete()

	ctx := context.Background()

	peerInfo := testPeers.GetTestPeerInfo(0)
	book := makeFakeLogbook(ctx, t, "test_user", peerInfo.PrivKey)

	// Dsrefs has a third reference that is not in logbook.
	dsrefs := []reporef.DatasetRef{
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.IDFromPeerID(peerInfo.PeerID),
			Name:      "first_new_name",
			Path:      "QmHashOfVersion2",
			FSIPath:   "/path/to/first_workspace",
		},
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.IDFromPeerID(peerInfo.PeerID),
			Name:      "second_name",
			Path:      "QmHashOfVersion6",
			FSIPath:   "/path/to/second_workspace",
		},
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.IDFromPeerID(peerInfo.PeerID),
			Name:      "third_name",
			Path:      "QmHashOfVersion100",
			FSIPath:   "/path/to/third_workspace",
		},
	}

	entryInfoList, err := convertLogbookAndRefs(ctx, book, dsrefs)
	if err != nil {
		t.Fatal(err)
	}

	expect := []*entryInfo{
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "htkkr2g4st3atjmxhkar3kjpv6x3xgls7sdkh4rm424v45tqpt6q",
				ProfileID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Name:      "first_new_name",
				Path:      "QmHashOfVersion2",
				FSIPath:   "/path/to/first_workspace",
			},
			TopIndex:    2,
			CursorIndex: 2,
		},
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "7n6dyt5aabo6j4fl2dbwwymoznsnd255egn6rb5cwchwetsoowzq",
				ProfileID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Name:      "second_name",
				Path:      "QmHashOfVersion6",
				FSIPath:   "/path/to/second_workspace",
			},
			TopIndex:    3,
			CursorIndex: 3,
		},
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				ProfileID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Name:      "third_name",
				Path:      "QmHashOfVersion100",
				FSIPath:   "/path/to/third_workspace",
			},
		},
	}
	if diff := cmp.Diff(expect, entryInfoList); diff != "" {
		t.Errorf("convertLogbookAndRefs (-want +got):\n%s", diff)
	}
}

// Test a logbook which has a dataset with no history, and a deleted dataset
func TestConvertLogbookAndRefsWithNoHistoryDatasetAndDeletedDataset(t *testing.T) {
	run := NewDscacheTestRunner()
	defer run.Delete()

	ctx := context.Background()

	peerInfo := testPeers.GetTestPeerInfo(0)
	book := makeFakeLogbookWithNoHistoryAndDelete(ctx, t, "test_user", peerInfo.PrivKey)

	// Dsrefs: first_ds is not checked out, second_ds was deleted, third_ds has no history
	dsrefs := []reporef.DatasetRef{
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.IDFromPeerID(peerInfo.PeerID),
			Name:      "first_ds",
			Path:      "QmHashOfVersion1001",
		},
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.IDFromPeerID(peerInfo.PeerID),
			Name:      "third_ds",
			FSIPath:   "/path/to/third_workspace",
		},
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.IDFromPeerID(peerInfo.PeerID),
			Name:      "fourth_ds",
			Path:      "QmHashOfVersion1005",
			FSIPath:   "/path/to/fourth_workspace",
		},
	}

	entryInfoList, err := convertLogbookAndRefs(ctx, book, dsrefs)
	if err != nil {
		t.Fatal(err)
	}

	expect := []*entryInfo{
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "3cf4zxzyxug7c2xmheltmnn3smnr3urpcifeyke4or7zunetu4ia",
				ProfileID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Name:      "first_ds",
				Path:      "QmHashOfVersion1001",
			},
			TopIndex:    1,
			CursorIndex: 1,
		},
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "n6bxzf53b3g4gugtn7svgpz2xmmbxp5ls6witdilt7oh5dtdnxwa",
				ProfileID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Name:      "third_ds",
				FSIPath:   "/path/to/third_workspace",
			},
		},
		&entryInfo{
			VersionInfo: dsref.VersionInfo{
				InitID:    "52iu62kxcgix5w7a5vwclf26gmxojnx67dnsddamkxokx7lxisnq",
				ProfileID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Name:      "fourth_ds",
				Path:      "QmHashOfVersion1005",
				FSIPath:   "/path/to/fourth_workspace",
			},
			TopIndex:    2,
			CursorIndex: 2,
		},
	}
	if diff := cmp.Diff(expect, entryInfoList); diff != "" {
		t.Errorf("convertLogbookAndRefs (-want +got):\n%s", diff)
	}
}

// Test the top-level build function, and that the references are alphabetized
func TestBuildDscacheFromLogbookAndProfilesAndDsrefAlphabetized(t *testing.T) {
	run := NewDscacheTestRunner()
	defer run.Delete()

	ctx := context.Background()

	peerInfo := testPeers.GetTestPeerInfo(0)
	book := makeFakeLogbookNonAlphabetical(ctx, t, "test_user", peerInfo.PrivKey)

	// Add stub defs, so that fillInfoForDatasets succeeds
	store := cafs.NewMapstore()
	run.MustPutDatasetFileAtKey(t, store, "/map/QmHashOfVersion1", `{}`)
	run.MustPutDatasetFileAtKey(t, store, "/map/QmHashOfVersion2", `{}`)
	run.MustPutDatasetFileAtKey(t, store, "/map/QmHashOfVersion3", `{}`)

	// Add association between profileID and username
	profiles := profile.NewMemStore()
	profiles.PutProfile(&profile.Profile{
		ID:       profile.IDFromPeerID(peerInfo.PeerID),
		Peername: "test_user",
	})

	dsrefs := []reporef.DatasetRef{}
	fs := qfs.NewMemFS()
	cache, err := BuildDscacheFromLogbookAndProfilesAndDsref(ctx, dsrefs, profiles, book, store, fs)
	if err != nil {
		t.Fatal(err)
	}

	refs, err := cache.ListRefs()
	if err != nil {
		t.Fatal(err)
	}

	// Verify that there are 3 datasets and that they are alphabetized.
	expectRefs := []string{
		"test_user/another_dataset@QmHashOfVersion2",
		"test_user/some_dataset@QmHashOfVersion1",
		"test_user/yet_another@QmHashOfVersion3",
	}
	if len(expectRefs) != len(refs) {
		t.Fatalf("expected: %d refs, got %d", len(expectRefs), len(refs))
	}
	for i, actualRef := range refs {
		if expectRefs[i] != actualRef.String() {
			t.Errorf("ref %d: expected %s, got %s", i, expectRefs[i], actualRef.String())
		}
	}
}

// Test that building dscache will fill info from datasets
func TestBuildDscacheFromLogbookAndProfilesAndDsrefFillInfo(t *testing.T) {
	run := NewDscacheTestRunner()
	defer run.Delete()

	ctx := context.Background()

	peerInfo := testPeers.GetTestPeerInfo(0)
	book := makeFakeLogbook(ctx, t, "test_user", peerInfo.PrivKey)

	// Add test datasets, which fillInfoForDatasets will use to populate dscache
	store := cafs.NewMapstore()
	run.MustPutDatasetFileAtKey(t, store, "/map/QmHashOfVersion2", `{
  "meta": {
    "title": "This Is Title",
    "theme": [
      "testdata",
      "example"
    ]
  }
}`)
	run.MustPutDatasetFileAtKey(t, store, "/map/QmHashOfVersion6", `{
  "structure": {
    "entries": 10,
    "length": 678,
    "errCount": 3
  },
  "commit": {
    "title": "Yet another commit",
    "message": "This is the third commit"
  }
}`)

	// Add association between profileID and username
	profiles := profile.NewMemStore()
	profiles.PutProfile(&profile.Profile{
		ID:       profile.IDFromPeerID(peerInfo.PeerID),
		Peername: "test_user",
	})

	dsrefs := []reporef.DatasetRef{}
	fs := qfs.NewMemFS()
	cache, err := BuildDscacheFromLogbookAndProfilesAndDsref(ctx, dsrefs, profiles, book, store, fs)
	if err != nil {
		t.Fatal(err)
	}

	actual := cache.VerboseString(false)
	expect := `Dscache:
 Dscache.Users:
  0) user=test_user profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = htkkr2g4st3atjmxhkar3kjpv6x3xgls7sdkh4rm424v45tqpt6q
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 2
     cursorIndex   = 2
     prettyName    = first_new_name
     metaTitle     = This Is Title
     themeList     = testdata,example
     commitTime    = -62135596800
     headRef       = QmHashOfVersion2
  1) initID        = 7n6dyt5aabo6j4fl2dbwwymoznsnd255egn6rb5cwchwetsoowzq
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 3
     cursorIndex   = 3
     prettyName    = second_name
     bodySize      = 678
     bodyRows      = 10
     commitTime    = -62135596800
     commitTitle   = Yet another commit
     commitMessage = This is the third commit
     numErrors     = 3
     headRef       = QmHashOfVersion6
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

// TODO(dlong): Test convertHistoryToIndexAndRef edge-cases, like big deletes, len(logs) > 0, len=0
// TODO(dlong): Test a logbook where a username is changed after datasets already existed
// TODO(dlong): Test a logbook with logs from other peers
// TODO(dlong): Test the function fillInfoForDatasets after convertLogbookAndRefs returns []entryInfo

func makeFakeLogbook(ctx context.Context, t *testing.T, username string, privKey crypto.PrivKey) *logbook.Book {
	rootPath, err := ioutil.TempDir("", "create_logbook")
	if err != nil {
		t.Fatal(err)
	}
	fs := qfs.NewMemFS()

	builder := NewLogbookTempBuilder(t, privKey, username, fs, rootPath)

	// A dataset with one commit, then a rename, then another commit
	refA := builder.DatasetInit(ctx, t, "first_name")
	refA = builder.Commit(ctx, t, refA, "initial commit", "QmHashOfVersion1")
	refA = builder.DatasetRename(ctx, t, refA, "first_new_name")
	refA = builder.Commit(ctx, t, refA, "another commit", "QmHashOfVersion2")

	// A dataset with five commits, two of which were deleted
	refB := builder.DatasetInit(ctx, t, "second_name")
	refB = builder.Commit(ctx, t, refB, "initial commit", "QmHashOfVersion3")
	refB = builder.Commit(ctx, t, refB, "second commit", "QmHashOfVersion4")
	refB = builder.Delete(ctx, t, refB, 1)
	refB = builder.Commit(ctx, t, refB, "fix second", "QmHashOfVersion5")
	refB = builder.Commit(ctx, t, refB, "third commit", "QmHashOfVersion6")
	refB = builder.Commit(ctx, t, refB, "whoops", "QmHashOfVersion7")
	refB = builder.Delete(ctx, t, refB, 1)

	return builder.Logbook()
}

func makeFakeLogbookNonAlphabetical(ctx context.Context, t *testing.T, username string, privKey crypto.PrivKey) *logbook.Book {
	rootPath, err := ioutil.TempDir("", "create_logbook")
	if err != nil {
		t.Fatal(err)
	}
	fs := qfs.NewMemFS()

	builder := NewLogbookTempBuilder(t, privKey, username, fs, rootPath)

	// A dataset with one commit
	refA := builder.DatasetInit(ctx, t, "some_dataset")
	refA = builder.Commit(ctx, t, refA, "initial commit", "QmHashOfVersion1")

	// Another dataset with one commit
	refB := builder.DatasetInit(ctx, t, "another_dataset")
	refB = builder.Commit(ctx, t, refB, "initial commit", "QmHashOfVersion2")

	// Yet another dataset with one commit
	refC := builder.DatasetInit(ctx, t, "yet_another")
	refC = builder.Commit(ctx, t, refC, "initial commit", "QmHashOfVersion3")

	return builder.Logbook()
}

func makeFakeLogbookWithNoHistoryAndDelete(ctx context.Context, t *testing.T, username string, privKey crypto.PrivKey) *logbook.Book {
	rootPath, err := ioutil.TempDir("", "logbook_nohist_delete")
	if err != nil {
		t.Fatal(err)
	}
	fs := qfs.NewMemFS()

	builder := NewLogbookTempBuilder(t, privKey, username, fs, rootPath)

	// A dataset with one commit, pretty normal. Corresponding dsref has no FSIPath (no checkout)
	refA := builder.DatasetInit(ctx, t, "first_ds")
	refA = builder.Commit(ctx, t, refA, "initial commit", "QmHashOfVersion1001")

	// A dataset with two commits that gets deleted
	refB := builder.DatasetInit(ctx, t, "second_ds")
	refB = builder.Commit(ctx, t, refB, "initial commit", "QmHashOfVersion1002")
	refB = builder.Commit(ctx, t, refB, "another commit", "QmHashOfVersion1003")
	builder.DatasetDelete(ctx, t, refB)

	// A dataset with no commits, hits the "no history" codepath
	_ = builder.DatasetInit(ctx, t, "third_ds")

	// A dataset with two commits, pretty normal
	refD := builder.DatasetInit(ctx, t, "fourth_ds")
	refD = builder.Commit(ctx, t, refD, "initial commit", "QmHashOfVersion1004")
	refD = builder.Commit(ctx, t, refD, "another commit", "QmHashOfVersion1005")

	return builder.Logbook()
}
