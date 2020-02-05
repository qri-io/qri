package dscache

import (
	"context"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	dscachefb "github.com/qri-io/qri/dscache/dscachefb"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// BuildDscacheFromLogbookAndProfilesAndDsref creates a dscache, building it from logbook and
// profiles and dsrefs.
// Deprecated: Dsref is going away once dscache is in use. For now, only FSIPath is retrieved
// from dsref, but in the future it will be added directly to dscache, with the file systems's
// linkfiles (.qri-ref) acting as the authoritative source.
func BuildDscacheFromLogbookAndProfilesAndDsref(ctx context.Context, refs []reporef.DatasetRef, profiles profile.Store, book *logbook.Book, store cafs.Filestore, filesys qfs.Filesystem) (*Dscache, error) {
	profileList, err := profiles.List()
	if err != nil {
		return nil, err
	}

	userProfileList := make([]userProfilePair, 0, len(profileList))
	for id, pro := range profileList {
		pair := userProfilePair{Username: pro.Peername, ProfileID: id.String()}
		userProfileList = append(userProfileList, pair)
	}

	// Convert logbook into dataset info list. Iterate refs to get FSI paths and anything
	// missing from logbook.
	dsInfoList, err := convertLogbookAndRefs(ctx, book, refs)
	if err != nil {
		return nil, err
	}

	err = fillInfoForDatasets(ctx, store, filesys, dsInfoList)
	if err != nil {
		log.Errorf("%s", err)
	}

	return buildDscacheFlatbuffer(userProfileList, dsInfoList), nil
}

type userProfilePair struct {
	Username  string
	ProfileID string
}

// buildDscacheFlatbuffer constructs the flatbuffer from the users and refs
func buildDscacheFlatbuffer(userPairList []userProfilePair, dsInfoList []*dsInfo) *Dscache {
	builder := flatbuffers.NewBuilder(0)

	// Construct user associations, between human-readable usernames and profileIDs
	userList := make([]flatbuffers.UOffsetT, 0, len(userPairList))
	for _, up := range userPairList {
		username := builder.CreateString(up.Username)
		profileID := builder.CreateString(up.ProfileID)
		dscachefb.UserAssocStart(builder)
		dscachefb.UserAssocAddUsername(builder, username)
		dscachefb.UserAssocAddProfileID(builder, profileID)
		userAssoc := dscachefb.UserAssocEnd(builder)
		userList = append(userList, userAssoc)
	}

	// Build users vector, iterating backwards due to using prepend
	dscachefb.DscacheStartUsersVector(builder, len(userList))
	for i := len(userList) - 1; i >= 0; i-- {
		u := userList[i]
		builder.PrependUOffsetT(u)
	}
	users := builder.EndVector(len(userList))

	// Construct refs, with all pertinent information for each dataset ref
	refList := make([]flatbuffers.UOffsetT, 0, len(dsInfoList))
	for _, ds := range dsInfoList {
		initID := builder.CreateString(ds.InitID)
		profileID := builder.CreateString(ds.ProfileID)
		prettyName := builder.CreateString(ds.PrettyName)
		metaTitle := builder.CreateString(ds.MetaTitle)
		themeList := builder.CreateString(ds.ThemeList)
		hashRef := builder.CreateString(ds.HeadRef)
		fsiPath := builder.CreateString(ds.FSIPath)
		dscachefb.RefCacheStart(builder)
		dscachefb.RefCacheAddInitID(builder, initID)
		dscachefb.RefCacheAddProfileID(builder, profileID)
		dscachefb.RefCacheAddTopIndex(builder, int32(ds.TopIndex))
		dscachefb.RefCacheAddCursorIndex(builder, int32(ds.CursorIndex))
		dscachefb.RefCacheAddPrettyName(builder, prettyName)
		dscachefb.RefCacheAddMetaTitle(builder, metaTitle)
		dscachefb.RefCacheAddThemeList(builder, themeList)
		dscachefb.RefCacheAddBodySize(builder, int64(ds.BodySize))
		dscachefb.RefCacheAddBodyRows(builder, int32(ds.BodyRows))
		dscachefb.RefCacheAddCommitTime(builder, ds.CommitTime.Unix())
		dscachefb.RefCacheAddNumErrors(builder, int32(ds.NumErrors))
		dscachefb.RefCacheAddHeadRef(builder, hashRef)
		dscachefb.RefCacheAddFsiPath(builder, fsiPath)
		ref := dscachefb.RefCacheEnd(builder)
		refList = append(refList, ref)
	}

	// Build refs vector, iterating backwards due to using prepend
	dscachefb.DscacheStartRefsVector(builder, len(dsInfoList))
	for i := len(refList) - 1; i >= 0; i-- {
		r := refList[i]
		builder.PrependUOffsetT(r)
	}
	refs := builder.EndVector(len(dsInfoList))

	// Construct top-level dscache
	dscachefb.DscacheStart(builder)
	dscachefb.DscacheAddUsers(builder, users)
	dscachefb.DscacheAddRefs(builder, refs)
	cache := dscachefb.DscacheEnd(builder)

	builder.Finish(cache)
	serialized := builder.FinishedBytes()
	root := dscachefb.GetRootAsDscache(serialized, 0)
	return &Dscache{Root: root, Buffer: serialized}
}

type dsInfo struct {
	InitID      string
	ProfileID   string
	TopIndex    int
	CursorIndex int
	PrettyName  string
	MetaTitle   string
	ThemeList   string
	BodySize    int64
	BodyRows    int
	CommitTime  time.Time
	NumErrors   int
	HeadRef     string
	FSIPath     string
}

// convertLogbookAndRefs builds dsInfo from each dataset in the logbook, plus FSIPath from old dsrefs
func convertLogbookAndRefs(ctx context.Context, book *logbook.Book, dsrefs []reporef.DatasetRef) ([]*dsInfo, error) {
	userLogs, err := book.ListAllLogs(ctx)
	if err != nil {
		return nil, err
	}

	allInfoList := make([]*dsInfo, 0)
	for _, userLog := range userLogs {
		if len(userLog.Ops) < 1 {
			log.Debugf("empty operation list found for user, cannot proceed")
			continue
		}
		// TODO(dlong): Test for username changes
		profileID := userLog.Ops[0].AuthorID
		// Get the info for each dataset in this user's collection.
		infoList := convertLogbookUserToDsInfoList(profileID, userLog.Logs)
		allInfoList = append(allInfoList, infoList...)
	}

	// Iterate dsrefs, add FSIPaths and any refs that are missing from logbook
	missingInfoList := make([]*dsInfo, 0)
	for _, ref := range dsrefs {
		info := findMatchingInfo(ref, allInfoList)
		if info != nil {
			info.FSIPath = ref.FSIPath
			continue
		}
		missingInfoList = append(missingInfoList, &dsInfo{
			ProfileID:  ref.ProfileID.String(),
			PrettyName: ref.Name,
			HeadRef:    ref.Path,
			FSIPath:    ref.FSIPath,
		})
	}
	// Append any missing dsInfos
	if len(missingInfoList) > 0 {
		allInfoList = append(allInfoList, missingInfoList...)
	}
	return allInfoList, nil
}

func convertLogbookUserToDsInfoList(profileID string, dsLogs []*oplog.Log) []*dsInfo {
	infoList := make([]*dsInfo, 0, len(dsLogs))
	for _, dsLog := range dsLogs {
		info := convertDatasetHistoryToDsInfo(*dsLog)
		if info == nil {
			continue
		}
		info.ProfileID = profileID
		infoList = append(infoList, info)
	}
	return infoList
}

func convertDatasetHistoryToDsInfo(dsLog oplog.Log) *dsInfo {
	// Get the final pretty name, most recently ammended.
	prettyName := ""
	for _, op := range dsLog.Ops {
		if op.Model != logbook.DatasetModel {
			log.Errorf("expected to be at the dataset level, got model number %d", op.Model)
			return nil
		}
		prettyName = op.Name
		if op.Type == oplog.OpTypeRemove {
			// Dataset ends its history by being deleted.
			return nil
		}
	}

	// Get the init-id here, because this the log for the dataset model.
	initID := dsLog.ID()
	if len(dsLog.Logs) != 1 {
		log.Errorf("expected only 1 branch, got %d\n", len(dsLog.Logs))
		return nil
	}

	historyLog := dsLog.Logs[0]
	topIndex, headRef := convertHistoryToIndexAndRef(*historyLog)
	cursorIndex := topIndex
	return &dsInfo{
		InitID:      initID,
		TopIndex:    topIndex,
		CursorIndex: cursorIndex,
		PrettyName:  prettyName,
		HeadRef:     headRef,
	}
}

func convertHistoryToIndexAndRef(historyLog oplog.Log) (int, string) {
	refs := make([]string, 0, len(historyLog.Ops))
	// Collect references added and removed to get those that remain.
	for _, op := range historyLog.Ops {
		if op.Type == oplog.OpTypeRemove {
			refs = refs[0 : len(refs)-int(op.Size)]
		} else {
			refs = append(refs, op.Ref)
		}
	}

	// Recursion should end at the branch/commit level, should not be any more sub-levels of logs.
	if len(historyLog.Logs) > 0 {
		log.Errorf("expected no more logs, has %d logs\n", len(historyLog.Logs))
	}

	// Get the last reference, treat this as top and cursor.
	lastIndex := len(refs) - 1
	lastRef := refs[lastIndex]
	return lastIndex, lastRef
}

func findMatchingInfo(ref reporef.DatasetRef, dsInfoList []*dsInfo) *dsInfo {
	for _, info := range dsInfoList {
		if info == nil {
			continue
		}
		// NOTE: This is a bad example of how to find a dataset, and should not be followed in
		// other code. Comparing ProfileID is okay, because they are stable ids that don't change,
		// but name is mutable and can be modified at any time. It should not be used as a
		// primary key, only as for pretty display. We should be using initID everywhere instead,
		// but dsrefs does not store the initID, which is the whole reason it is going away.
		if ref.ProfileID.String() == info.ProfileID && ref.Name == info.PrettyName {
			return info
		}
	}
	return nil
}
