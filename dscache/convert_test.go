package dscache

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qfs"
	testPeers "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/logbook"
	reporef "github.com/qri-io/qri/repo/ref"
	"github.com/qri-io/qri/repo/profile"
)

// Test the buildDscacheFlatbuffer function, which converts plain-old-data structures into dscache
func TestBuildDscacheFlatbuffer(t *testing.T) {
	pid0 := profile.ID(testPeers.GetTestPeerInfo(0).PeerID)
	pid1 := profile.ID(testPeers.GetTestPeerInfo(1).PeerID)
	pid2 := profile.ID(testPeers.GetTestPeerInfo(2).PeerID)

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
	dsInfoList := []*dsInfo{
		&dsInfo{
			InitID:     "ds_init_id_0000",
			ProfileID:  pid1.String(),
			PrettyName: "my_ds",
			HeadRef:    "/ipfs/QmExampleFirst",
		},
		&dsInfo{
			InitID:     "ds_init_id_0001",
			ProfileID:  pid1.String(),
			PrettyName: "another_ds",
			HeadRef:    "/ipfs/QmExampleSecond",
		},
		&dsInfo{
			InitID:     "ds_init_id_0002",
			ProfileID:  pid1.String(),
			PrettyName: "checked_out_ds",
			HeadRef:    "/ipfs/QmExampleThird",
			FSIPath:    "/path/to/workspace/checked_out_ds",
		},
		&dsInfo{
			InitID:     "ds_init_id_0003",
			ProfileID:  pid2.String(),
			PrettyName: "foreign_ds",
			HeadRef:    "/ipfs/QmExampleFourth",
		},
	}
	dscache := buildDscacheFlatbuffer(userList, dsInfoList)
	actual := dscache.VerboseString(false)

	expect := `Dscache:
 Dscache.Users:
  0) user=test_zero profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  1) user=test_one profileID=QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
  2) user=test_two profileID=QmPeFTNHcZDr3ZFEfFfepxS5PqHAmfBRGQNPJ389Cwh1as
 Dscache.Refs:
  0) initID      = ds_init_id_0000
     profileID   = QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
     topIndex    = 0
     cursorIndex = 0
     prettyName  = my_ds
     commitTime  = -62135596800
     headRef     = /ipfs/QmExampleFirst
  1) initID      = ds_init_id_0001
     profileID   = QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
     topIndex    = 0
     cursorIndex = 0
     prettyName  = another_ds
     commitTime  = -62135596800
     headRef     = /ipfs/QmExampleSecond
  2) initID      = ds_init_id_0002
     profileID   = QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
     topIndex    = 0
     cursorIndex = 0
     prettyName  = checked_out_ds
     commitTime  = -62135596800
     headRef     = /ipfs/QmExampleThird
     fsiPath     = /path/to/workspace/checked_out_ds
  3) initID      = ds_init_id_0003
     profileID   = QmPeFTNHcZDr3ZFEfFfepxS5PqHAmfBRGQNPJ389Cwh1as
     topIndex    = 0
     cursorIndex = 0
     prettyName  = foreign_ds
     commitTime  = -62135596800
     headRef     = /ipfs/QmExampleFourth
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

// Test the convertLogbookAndRefs function, which turns a logbook and dsrefs into a list of dsInfo
func TestConvertLogbookAndRefsBasic(t *testing.T) {
	run := NewDscacheTestRunner()
	defer run.Delete()

	ctx := context.Background()

	peerInfo := testPeers.GetTestPeerInfo(0)
	book := makeFakeLogbook(ctx, t, "test_user", peerInfo.PrivKey)

	dsrefs := []reporef.DatasetRef{
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.ID(peerInfo.PeerID),
			Name:      "first_new_name",
			Path:      "QmHashOfVersion2",
			FSIPath:   "/path/to/first_workspace",
		},
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.ID(peerInfo.PeerID),
			Name:      "second_name",
			Path:      "QmHashOfVersion6",
			FSIPath:   "/path/to/second_workspace",
		},
	}

	dsInfoList, err := convertLogbookAndRefs(ctx, book, dsrefs)
	if err != nil {
		t.Fatal(err)
	}

	expect := []*dsInfo{
		&dsInfo{
			InitID:      "htkkr2g4st3atjmxhkar3kjpv6x3xgls7sdkh4rm424v45tqpt6q",
			ProfileID:   "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
			TopIndex:    2,
			CursorIndex: 2,
			PrettyName:  "first_new_name",
			HeadRef:     "QmHashOfVersion2",
			FSIPath:     "/path/to/first_workspace",
		},
		&dsInfo{
			InitID:      "7n6dyt5aabo6j4fl2dbwwymoznsnd255egn6rb5cwchwetsoowzq",
			ProfileID:   "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
			TopIndex:    3,
			CursorIndex: 3,
			PrettyName:  "second_name",
			HeadRef:     "QmHashOfVersion6",
			FSIPath:     "/path/to/second_workspace",
		},
	}
	if diff := cmp.Diff(expect, dsInfoList); diff != "" {
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
			ProfileID: profile.ID(peerInfo.PeerID),
			Name:      "first_new_name",
			Path:      "QmHashOfVersion2",
			FSIPath:   "/path/to/first_workspace",
		},
	}

	dsInfoList, err := convertLogbookAndRefs(ctx, book, dsrefs)
	if err != nil {
		t.Fatal(err)
	}

	expect := []*dsInfo{
		&dsInfo{
			InitID:      "htkkr2g4st3atjmxhkar3kjpv6x3xgls7sdkh4rm424v45tqpt6q",
			ProfileID:   "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
			TopIndex:    2,
			CursorIndex: 2,
			PrettyName:  "first_new_name",
			HeadRef:     "QmHashOfVersion2",
			FSIPath:     "/path/to/first_workspace",
		},
		&dsInfo{
			InitID:      "7n6dyt5aabo6j4fl2dbwwymoznsnd255egn6rb5cwchwetsoowzq",
			ProfileID:   "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
			TopIndex:    3,
			CursorIndex: 3,
			PrettyName:  "second_name",
			HeadRef:     "QmHashOfVersion6",
		},
	}
	if diff := cmp.Diff(expect, dsInfoList); diff != "" {
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
			ProfileID: profile.ID(peerInfo.PeerID),
			Name:      "first_new_name",
			Path:      "QmHashOfVersion2",
			FSIPath:   "/path/to/first_workspace",
		},
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.ID(peerInfo.PeerID),
			Name:      "second_name",
			Path:      "QmHashOfVersion6",
			FSIPath:   "/path/to/second_workspace",
		},
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.ID(peerInfo.PeerID),
			Name:      "third_name",
			Path:      "QmHashOfVersion100",
			FSIPath:   "/path/to/third_workspace",
		},
	}

	dsInfoList, err := convertLogbookAndRefs(ctx, book, dsrefs)
	if err != nil {
		t.Fatal(err)
	}

	expect := []*dsInfo{
		&dsInfo{
			InitID:      "htkkr2g4st3atjmxhkar3kjpv6x3xgls7sdkh4rm424v45tqpt6q",
			ProfileID:   "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
			TopIndex:    2,
			CursorIndex: 2,
			PrettyName:  "first_new_name",
			HeadRef:     "QmHashOfVersion2",
			FSIPath:     "/path/to/first_workspace",
		},
		&dsInfo{
			InitID:      "7n6dyt5aabo6j4fl2dbwwymoznsnd255egn6rb5cwchwetsoowzq",
			ProfileID:   "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
			TopIndex:    3,
			CursorIndex: 3,
			PrettyName:  "second_name",
			HeadRef:     "QmHashOfVersion6",
			FSIPath:     "/path/to/second_workspace",
		},
		&dsInfo{
			ProfileID:   "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
			PrettyName:  "third_name",
			HeadRef:     "QmHashOfVersion100",
			FSIPath:     "/path/to/third_workspace",
		},
	}
	if diff := cmp.Diff(expect, dsInfoList); diff != "" {
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
			ProfileID: profile.ID(peerInfo.PeerID),
			Name:      "first_ds",
			Path:      "QmHashOfVersion1001",
		},
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.ID(peerInfo.PeerID),
			Name:      "third_ds",
			FSIPath:   "/path/to/third_workspace",
		},
		reporef.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.ID(peerInfo.PeerID),
			Name:      "fourth_ds",
			Path:      "QmHashOfVersion1005",
			FSIPath:   "/path/to/fourth_workspace",
		},
	}

	dsInfoList, err := convertLogbookAndRefs(ctx, book, dsrefs)
	if err != nil {
		t.Fatal(err)
	}

	expect := []*dsInfo{
		&dsInfo{
			InitID:      "3cf4zxzyxug7c2xmheltmnn3smnr3urpcifeyke4or7zunetu4ia",
			ProfileID:   "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
			TopIndex:    1,
			CursorIndex: 1,
			PrettyName:  "first_ds",
			HeadRef:     "QmHashOfVersion1001",
		},
		&dsInfo{
			InitID:      "n6bxzf53b3g4gugtn7svgpz2xmmbxp5ls6witdilt7oh5dtdnxwa",
			ProfileID:   "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
			PrettyName:  "third_ds",
			FSIPath:     "/path/to/third_workspace",
		},
		&dsInfo{
			InitID:      "52iu62kxcgix5w7a5vwclf26gmxojnx67dnsddamkxokx7lxisnq",
			ProfileID:   "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
			TopIndex:    2,
			CursorIndex: 2,
			PrettyName:  "fourth_ds",
			HeadRef:     "QmHashOfVersion1005",
			FSIPath:     "/path/to/fourth_workspace",
		},
	}
	if diff := cmp.Diff(expect, dsInfoList); diff != "" {
		t.Errorf("convertLogbookAndRefs (-want +got):\n%s", diff)
	}
}

// TODO(dlong): Test BuildDscacheFromLogbookAndProfilesAndDsref, the top-level function
// TODO(dlong): Test convertHistoryToIndexAndRef edge-cases, like big deletes, len(logs) > 0, len=0
// TODO(dlong): Test a logbook where a username is changed after datasets already existed
// TODO(dlong): Test the function fillInfoForDatasets after convertLogbookAndRefs returns []dsInfo

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
