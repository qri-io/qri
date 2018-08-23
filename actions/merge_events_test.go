package actions

import (
	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	"testing"
)

const refPath0 = "/ipfs/Qmaau1d1WjnQdTYfRYfFVsCS97cgD8ATyrKuNoEfexL7JZ"
const refPath1 = "/ipfs/QmafgXF3u3QSWErQoZ2URmQp5PFG384htoE7J338nS2H7T"
const refPath2 = "/ipfs/QmbNinL4ErzM73BxQSNf8q2vPmAM4v3MNQM2DmDqDjt47D"
const refPath3 = "/ipfs/Qmc5do1bC3JH73MThEgKgKkgNKLmqrvh2uaE6919yDmUaa"

const peerAID = "QmenK8PgcpM2KYzEKnGx1LyN1QXawswM2F6HktCbWKuC1b"
const peerBID = "Qmf2fBLzCvFxs3vvZaoQyKSTv2rjApek4F1kzRxAfK6P4P"

const profileAID = "QmSMZwvs3n4GBikZ7hKzT17c3bk65FmyZo2JqtJ16swqCv"
const profileBID = "QmSk46nSD78YiuNojYMS8NW6hSCjH95JajqqzzoychZgef"

func createReposAndLogs() (repo.Repo, repo.Repo, *repo.MemEventLog, *repo.MemEventLog) {
	aRepo, _ := repo.NewMemRepo(&profile.Profile{
		ID:       profile.ID(profileAID),
		Peername: "test-peer-0",
	}, cafs.NewMapstore(), profile.MemStore{}, nil)
	bRepo, _ := repo.NewMemRepo(&profile.Profile{
		ID:       profile.ID(profileBID),
		Peername: "test-peer-0",
	}, cafs.NewMapstore(), profile.MemStore{}, nil)
	aLog := aRepo.MemEventLog
	bLog := bRepo.MemEventLog
	return aRepo, bRepo, aLog, bLog
}

func TestNewChangesThatCanMerge(t *testing.T) {
	aRepo, bRepo, aLog, bLog := createReposAndLogs()

	// Going to make 3 updates after the initial creation, but dataset name never changes.
	ref := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-0", Path: refPath0}
	ref1 := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-0", Path: refPath1}
	ref2 := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-0", Path: refPath2}
	ref3 := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-0", Path: refPath3}

	peerAID, _ := peer.IDB58Decode(peerAID)
	peerBID, _ := peer.IDB58Decode(peerBID)

	// Events for A.
	aLog.LogEventDetails(repo.ETDsCreated, 1000, peerAID, ref, nil)
	aLog.LogEventDetails(repo.ETDsPinned, 1001, peerAID, ref, nil)
	aLog.LogEventDetails(repo.ETDsCreated, 1002, peerAID, ref1, nil)
	// Events for B (same exact events).
	bLog.LogEventDetails(repo.ETDsCreated, 1000, peerAID, ref, nil)
	bLog.LogEventDetails(repo.ETDsPinned, 1001, peerAID, ref, nil)
	bLog.LogEventDetails(repo.ETDsCreated, 1002, peerAID, ref1, nil)
	// New stuff on B, can be merged.
	bLog.LogEventDetails(repo.ETDsCreated, 1010, peerBID, ref2, nil)
	bLog.LogEventDetails(repo.ETDsCreated, 1011, peerBID, ref3, nil)

	resultSet, err := MergeRepoEvents(aRepo, bRepo)
	if err != nil {
		log.Fatal(err)
	}
	if resultSet.Peer(0).NumConflicts() != 0 {
		t.Errorf("Expected no conflicts")
	}
	if resultSet.Peer(0).NumUpdates() != 2 {
		t.Errorf("Expected 2 updates for Peer A")
	}
	if resultSet.Peer(1).NumConflicts() != 0 {
		t.Errorf("Expected no conflicts")
	}
	if resultSet.Peer(1).NumUpdates() != 0 {
		t.Errorf("Expected 0 updates for Peer B")
	}
}

func TestBothMadeChanges(t *testing.T) {
	aRepo, bRepo, aLog, bLog := createReposAndLogs()

	// Going to make 1 update after the initial creation, then an update on each repo.
	ref := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-0", Path: refPath0}
	ref1 := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-0", Path: refPath1}
	ref2 := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-0", Path: refPath2}
	ref3 := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-0", Path: refPath3}

	peerAID, _ := peer.IDB58Decode(peerAID)
	peerBID, _ := peer.IDB58Decode(peerBID)

	// Events for A.
	aLog.LogEventDetails(repo.ETDsCreated, 1000, peerAID, ref, nil)
	aLog.LogEventDetails(repo.ETDsPinned, 1001, peerAID, ref, nil)
	aLog.LogEventDetails(repo.ETDsCreated, 1002, peerAID, ref1, nil)
	// Events for B (same exact events).
	bLog.LogEventDetails(repo.ETDsCreated, 1000, peerAID, ref, nil)
	bLog.LogEventDetails(repo.ETDsPinned, 1001, peerAID, ref, nil)
	bLog.LogEventDetails(repo.ETDsCreated, 1002, peerAID, ref1, nil)
	// New stuff on A.
	aLog.LogEventDetails(repo.ETDsCreated, 1010, peerAID, ref2, nil)
	// Also new stuff on B, this is a conflict.
	bLog.LogEventDetails(repo.ETDsCreated, 1020, peerBID, ref3, nil)

	resultSet, err := MergeRepoEvents(aRepo, bRepo)
	if err != nil {
		log.Fatal(err)
	}
	if resultSet.Peer(0).NumConflicts() != 1 {
		t.Errorf("Expected 1 conflict")
	}
	if resultSet.Peer(0).NumUpdates() != 0 {
		t.Errorf("Expected 0 updates for Peer A, got %d", resultSet.Peer(0).NumUpdates())
	}
	if resultSet.Peer(1).NumConflicts() != 1 {
		t.Errorf("Expected 1 conflict")
	}
	if resultSet.Peer(1).NumUpdates() != 0 {
		t.Errorf("Expected 0 updates for Peer B, got %d", resultSet.Peer(1).NumUpdates())
	}
}

func TestDeleteAfterRename(t *testing.T) {
	aRepo, bRepo, aLog, bLog := createReposAndLogs()

	peerAID, _ := peer.IDB58Decode(peerAID)
	peerBID, _ := peer.IDB58Decode(peerBID)

	// Going to make 1 update after the initial creation, which is a rename.
	ref := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-0", Path: refPath0}
	ref1 := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-1", Path: refPath0}

	// Events for A
	aLog.LogEventDetails(repo.ETDsCreated, 1000, peerAID, ref, nil)
	aLog.LogEventDetails(repo.ETDsPinned, 1001, peerAID, ref, nil)
	// Events for B (same exact events).
	bLog.LogEventDetails(repo.ETDsCreated, 1000, peerAID, ref, nil)
	bLog.LogEventDetails(repo.ETDsPinned, 1001, peerAID, ref, nil)

	// B renames.
	bLog.LogEventDetails(repo.ETDsRenamed, 1010, peerBID, ref1,
		[2]string{"test-dataset-0", "test-dataset-1"})
	// A deletes (should apply to new name).
	aLog.LogEventDetails(repo.ETDsDeleted, 1020, peerAID, ref, nil)

	resultSet, err := MergeRepoEvents(aRepo, bRepo)
	if err != nil {
		log.Fatal(err)
	}
	if resultSet.Peer(0).NumConflicts() != 0 {
		t.Errorf("Expected no conflicts")
	}
	// Maybe, Peer A doesn't have any updates because it deleted the dataset,
	// so doesn't need to do any work.
	if resultSet.Peer(0).NumUpdates() != 1 {
		t.Errorf("Expected 1 updates for Peer A")
	}
	if resultSet.Peer(1).NumConflicts() != 0 {
		t.Errorf("Expected no conflicts")
	}
	if resultSet.Peer(1).NumUpdates() != 1 {
		t.Errorf("Expected 1 updates for Peer B")
	}
}

func TestRenameAndAddContent(t *testing.T) {
	aRepo, bRepo, aLog, bLog := createReposAndLogs()

	peerAID, _ := peer.IDB58Decode(peerAID)
	peerBID, _ := peer.IDB58Decode(peerBID)

	// Going to make 1 update after the initial creation, which is a rename.
	ref := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-0", Path: refPath0}
	ref1 := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-1", Path: refPath0}
	ref2 := repo.DatasetRef{Peername: "test-peer-0", Name: "test-dataset-0", Path: refPath2}

	// Events for A
	aLog.LogEventDetails(repo.ETDsCreated, 1000, peerAID, ref, nil)
	aLog.LogEventDetails(repo.ETDsPinned, 1001, peerAID, ref, nil)
	// Events for B (same exact events).
	bLog.LogEventDetails(repo.ETDsCreated, 1000, peerAID, ref, nil)
	bLog.LogEventDetails(repo.ETDsPinned, 1001, peerAID, ref, nil)

	// B renames.
	bLog.LogEventDetails(repo.ETDsRenamed, 1010, peerBID, ref1,
		[2]string{"test-dataset-0", "test-dataset-1"})
	// A adds content.
	aLog.LogEventDetails(repo.ETDsCreated, 1020, peerAID, ref2, nil)

	resultSet, err := MergeRepoEvents(aRepo, bRepo)
	if err != nil {
		log.Fatal(err)
	}
	if resultSet.Peer(0).NumConflicts() != 0 {
		t.Errorf("Expected no conflicts")
	}
	if resultSet.Peer(0).NumUpdates() != 1 {
		t.Errorf("Expected 1 updates for Peer A")
	}
	if resultSet.Peer(1).NumConflicts() != 0 {
		t.Errorf("Expected no conflicts")
	}
	if resultSet.Peer(1).NumUpdates() != 1 {
		t.Errorf("Expected 1 updates for Peer B")
	}
}
