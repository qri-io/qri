package dscache

import (
	flatbuffers "github.com/google/flatbuffers/go"
	dscachefb "github.com/qri-io/qri/dscache/dscachefb"
)

func (d *Dscache) copyUserAssociationList(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	userList := make([]flatbuffers.UOffsetT, 0, d.Root.UsersLength())
	for i := 0; i < d.Root.UsersLength(); i++ {
		up := dscachefb.UserAssoc{}
		d.Root.Users(&up, i)
		d.copyUserAssoc(builder, &up)
		user := dscachefb.UserAssocEnd(builder)
		userList = append(userList, user)
	}
	dscachefb.DscacheStartUsersVector(builder, len(userList))
	for i := len(userList) - 1; i >= 0; i-- {
		u := userList[i]
		builder.PrependUOffsetT(u)
	}
	return builder.EndVector(len(userList))
}

func (d *Dscache) copyReferenceListWithReplacement(
	builder *flatbuffers.Builder,
	findMatchFunc func(*dscachefb.RefCache) bool,
	replaceRefFunc func(func(*flatbuffers.Builder))) flatbuffers.UOffsetT {

	// Construct refs, with all pertinent information for each dataset ref
	refList := make([]flatbuffers.UOffsetT, 0, d.Root.RefsLength())
	for i := 0; i < d.Root.RefsLength(); i++ {
		r := dscachefb.RefCache{}
		d.Root.Refs(&r, i)
		// Check if this entry is the one that we want to modify.
		if findMatchFunc(&r) {
			// This is due to the flatbuffers API being a bit awkward to use.
			// The replace func may want to create some slots (such as strings) before the
			// builder starts on construction. This means we can't call copyReference now, instead,
			// pass it as a func to the callback, let it start construction when it is ready.
			startRefBuildFunc := func(_ *flatbuffers.Builder) {
				d.copyReference(builder, &r)
			}
			replaceRefFunc(startRefBuildFunc)
			ref := dscachefb.RefCacheEnd(builder)
			refList = append(refList, ref)
			continue
		}
		d.copyReference(builder, &r)
		ref := dscachefb.RefCacheEnd(builder)
		refList = append(refList, ref)
	}
	dscachefb.DscacheStartRefsVector(builder, len(refList))
	for i := len(refList) - 1; i >= 0; i-- {
		r := refList[i]
		builder.PrependUOffsetT(r)
	}
	return builder.EndVector(len(refList))
}

func (d *Dscache) finishBuilding(builder *flatbuffers.Builder, users, refs flatbuffers.UOffsetT) (*dscachefb.Dscache, []byte) {
	dscachefb.DscacheStart(builder)
	dscachefb.DscacheAddUsers(builder, users)
	dscachefb.DscacheAddRefs(builder, refs)
	cache := dscachefb.DscacheEnd(builder)
	builder.Finish(cache)
	serialized := builder.FinishedBytes()
	return dscachefb.GetRootAsDscache(serialized, 0), serialized
}

func (d *Dscache) copyUserAssoc(builder *flatbuffers.Builder, ua *dscachefb.UserAssoc) {
	username := builder.CreateString(string(ua.Username()))
	profileID := builder.CreateString(string(ua.ProfileID()))
	dscachefb.UserAssocStart(builder)
	dscachefb.UserAssocAddUsername(builder, username)
	dscachefb.UserAssocAddProfileID(builder, profileID)
}

func (d *Dscache) copyReference(builder *flatbuffers.Builder, r *dscachefb.RefCache) {
	initID := builder.CreateString(string(r.InitID()))
	profileID := builder.CreateString(string(r.ProfileID()))
	prettyName := builder.CreateString(string(r.PrettyName()))
	metaTitle := builder.CreateString(string(r.MetaTitle()))
	themeList := builder.CreateString(string(r.ThemeList()))
	hashRef := builder.CreateString(string(r.HeadRef()))
	fsiPath := builder.CreateString(string(r.FsiPath()))
	dscachefb.RefCacheStart(builder)
	dscachefb.RefCacheAddInitID(builder, initID)
	dscachefb.RefCacheAddProfileID(builder, profileID)
	dscachefb.RefCacheAddTopIndex(builder, int32(r.TopIndex()))
	dscachefb.RefCacheAddCursorIndex(builder, int32(r.CursorIndex()))
	dscachefb.RefCacheAddPrettyName(builder, prettyName)
	dscachefb.RefCacheAddMetaTitle(builder, metaTitle)
	dscachefb.RefCacheAddThemeList(builder, themeList)
	dscachefb.RefCacheAddBodySize(builder, int64(r.BodySize()))
	dscachefb.RefCacheAddBodyRows(builder, int32(r.BodyRows()))
	dscachefb.RefCacheAddCommitTime(builder, r.CommitTime())
	dscachefb.RefCacheAddNumErrors(builder, int32(r.NumErrors()))
	dscachefb.RefCacheAddHeadRef(builder, hashRef)
	dscachefb.RefCacheAddFsiPath(builder, fsiPath)
}
