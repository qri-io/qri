package actions

import (
	"github.com/qri-io/qri/repo"
)

// MergeResultSet contains information about how to merge a collection of EventLogs.
type MergeResultSet struct {
	peers []MergeResultEntry
}

// MergeResultEntry contains information about how a single peer should update its EventLog.
type MergeResultEntry struct {
	conflicts int
	updates   int
}

// Peer gets a MegeResultEntry for a single peer.
func (s MergeResultSet) Peer(i int) MergeResultEntry {
	return s.peers[i]
}

// NumConflicts gets the number of conflicts.
func (e MergeResultEntry) NumConflicts() int {
	return e.conflicts
}

// NumUpdates gets the number of updates.
func (e MergeResultEntry) NumUpdates() int {
	return e.updates
}

// MergeRepoEvents tries to merge multiple EventLogs.
func MergeRepoEvents(one repo.Repo, two repo.Repo) (MergeResultSet, error) {
	resultSet := MergeResultSet{}
	resultSet.peers = make([]MergeResultEntry, 2)
	oneLog := one.(repo.EventLog)
	twoLog := two.(repo.EventLog)
	// TODO: Handle any length of EventLogs.
	oneEvents, _ := oneLog.Events(100, 0)
	twoEvents, _ := twoLog.Events(100, 0)
	var possibleConflictEvent *repo.Event
	// Events are stored in reverse timestamp order.
	i := len(oneEvents) - 1
	j := len(twoEvents) - 1
	for i >= 0 && j >= 0 {
		// TODO: It may be correct to find the first point of divergence, and then
		// check each pair-wise elements using CanResolveEvents. This loop, which works
		// like a zip, incorrectly assumes that a conflict can occur, followed by more
		// matching elements, but there's really no use case where that can happen.
		oneEv := oneEvents[i]
		twoEv := twoEvents[j]
		if oneEv.Time == twoEv.Time {
			i--
			j--
			continue
		} else if oneEv.Time.Before(twoEv.Time) {
			// TODO: Handle more than one conflict at a time.
			possibleConflictEvent = oneEv
			i--
			continue
		} else if twoEv.Time.Before(oneEv.Time) {
			possibleConflictEvent = twoEv
			j--
			continue
		}
	}
	// Handle any leftovers.
	for i >= 0 {
		oneEv := oneEvents[i]
		if possibleConflictEvent == nil {
			// TODO: Add data from oneEv into the update.
			resultSet.peers[1].updates++
		} else if CanResolveEvents(*possibleConflictEvent, *oneEv) {
			resultSet.peers[0].updates++
			resultSet.peers[1].updates++
		} else {
			resultSet.peers[0].conflicts++
			resultSet.peers[1].conflicts++
		}
		i--
	}
	for j >= 0 {
		twoEv := twoEvents[j]
		if possibleConflictEvent == nil {
			// TODO: Add data from twoEv into the update.
			resultSet.peers[0].updates++
		} else if CanResolveEvents(*possibleConflictEvent, *twoEv) {
			resultSet.peers[0].updates++
			resultSet.peers[1].updates++
		} else {
			resultSet.peers[0].conflicts++
			resultSet.peers[1].conflicts++
		}
		j--
	}
	return resultSet, nil
}

// CanResolveEvents determines whether two Events can be resolved, or if they conflict.
func CanResolveEvents(left repo.Event, right repo.Event) bool {
	// TODO: Handle more cases.
	if left.Type == repo.ETDsRenamed && right.Type == repo.ETDsRenamed {
		return false
	} else if left.Type == repo.ETDsRenamed || right.Type == repo.ETDsRenamed {
		return true
	} else {
		return false
	}
}
