package collection

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/repo"
)

// MigrateRepoStoreToLocalCollectionSet constructs a local collection.Set from
// legacy repo data
func MigrateRepoStoreToLocalCollectionSet(ctx context.Context, s Set, r repo.Repo) error {
	datasets, err := repo.ListVersionInfoShim(r, 0, 1000000)
	if err != nil {
		return err
	}

	// empty collection migration needs to create a set for the repo owner
	ownerID := r.Profiles().Owner(ctx).ID
	if len(datasets) == 0 {
		if ls, ok := s.(*localSet); ok {
			ls.collections[ownerID] = []dsref.VersionInfo{}
		}
	}

	book := r.Logbook()
	for i, vi := range datasets {
		ref := vi.SimpleRef()
		if _, err := book.ResolveRef(ctx, &ref); err != nil {
			log.Errorf("can't migrate dataset %s to collection. Error resolving dataset initID: %s", vi.SimpleRef(), err)
			continue
		}
		datasets[i].InitID = ref.InitID
		if ds, loadErr := dsfs.LoadDataset(ctx, r.Filesystem(), ref.Path); loadErr == nil {
			datasets[i].CommitTime = ds.Commit.Timestamp
			datasets[i].CommitTitle = ds.Commit.Title
			datasets[i].BodyRows = ds.Structure.Entries
			datasets[i].BodySize = ds.Structure.Length
			datasets[i].NumErrors = ds.Structure.ErrCount
			if ds.Meta != nil {
				datasets[i].MetaTitle = ds.Meta.Title
			}
		}

		if err := addRunAndCommitInfo(ctx, book, &datasets[i]); err != nil {
			log.Error(err)
			continue
		}
	}

	// remove any datasets that couldn't be resolved
	for i := len(datasets) - 1; i >= 0; i-- {
		if datasets[i].InitID == "" {
			datasets = append(datasets[:i], datasets[i+1:]...)
		}
	}

	return s.Add(ctx, ownerID, datasets...)
}

func addRunAndCommitInfo(ctx context.Context, book *logbook.Book, vi *dsref.VersionInfo) error {
	ulog, err := book.UserDatasetBranchesLog(ctx, vi.InitID)
	if err != nil {
		return fmt.Errorf("can't get user log for dataset %s, %w", vi.InitID, err)
	}
	dlog, err := ulog.Log(vi.InitID)
	if err != nil {
		return fmt.Errorf("can't get dataset log for dataset %s, %w", vi.InitID, err)
	}
	if len(dlog.Logs) < 0 {
		return fmt.Errorf("no branch logs for dataset log %s, %w", vi.InitID, err)
	}
	blog := dlog.Logs[0]
	commitCount := 0
	runCount := 0
	mostRecentRunRecorded := false
	for _, op := range blog.Ops {
		if op.Model == logbook.CommitModel {
			switch op.Type {
			case oplog.OpTypeInit:
				commitCount++
			case oplog.OpTypeAmend:
				continue
			case oplog.OpTypeRemove:
				commitCount = int(int64(commitCount) - op.Size)
			}
			continue
		}
		if op.Model == logbook.RunModel {
			runCount++
			if !mostRecentRunRecorded {
				vi.RunID = op.Ref
				vi.RunDuration = op.Size
				vi.RunStatus = op.Note
				mostRecentRunRecorded = true
			}
			continue
		}
	}
	vi.CommitCount = commitCount
	vi.RunCount = runCount
	return nil
}
