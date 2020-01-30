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
	"github.com/qri-io/qri/repo"
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
func TestConvertLogbookAndRefs(t *testing.T) {
	run := NewDscacheTestRunner()
	defer run.Delete()

	ctx := context.Background()

	peerInfo := testPeers.GetTestPeerInfo(0)
	book := makeFakeLogbook(ctx, t, "test_user", peerInfo.PrivKey)

	dsrefs := []repo.DatasetRef{
		repo.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.ID(peerInfo.PeerID),
			Name:      "first_new_name",
			FSIPath:   "/path/to/first_workspace",
		},
		repo.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.ID(peerInfo.PeerID),
			Name:      "second_name",
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
	dsrefs := []repo.DatasetRef{
		repo.DatasetRef{
			Peername:  "test_user",
			ProfileID: profile.ID(peerInfo.PeerID),
			Name:      "first_new_name",
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

// TODO(dlong): Test BuildDscacheFromLogbookAndProfilesAndDsref, the top-level function
// TODO(dlong): Test convertHistoryToIndexAndRef edge-cases, like big deletes, len(logs) > 0, len=0
// TODO(dlong): Test different fake logbooks aside from the one created by makeFakeLogbook
// TODO(dlong): Test a logbook where a username is changed after datasets already existed
// TODO(dlong): Test a logbook with a dataset that has no history (init but no commit)
// TODO(dlong): Test a dataset being deleted from a logbook
// TODO(dlong): Test the function fillInfoForDatasets after convertLogbookAndRefs returns []dsInfo

func makeFakeLogbook(ctx context.Context, t *testing.T, username string, privKey crypto.PrivKey) *logbook.Book {
	rootPath, err := ioutil.TempDir("", "create_logbook")
	if err != nil {
		t.Fatal(err)
	}
	fs := qfs.NewMemFS()

	builder := NewLogbookTempBuilder(t, privKey, username, fs, rootPath)

	refA := builder.DatasetInit(ctx, t, "first_name")
	refA = builder.Commit(ctx, t, refA, "initial commit", "QmHashOfVersion1")
	refA = builder.DatasetRename(ctx, t, refA, "first_new_name")
	refA = builder.Commit(ctx, t, refA, "another commit", "QmHashOfVersion2")

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
