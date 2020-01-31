package repo

import (
	flatbuffers "github.com/google/flatbuffers/go"
	repofb "github.com/qri-io/qri/repo/repo_fbs"
	reporef "github.com/qri-io/qri/repo/ref"
	"github.com/qri-io/qri/repo/profile"
)

// RefList is a list of refs
type RefList []reporef.DatasetRef

// Len returns the length of refs. Used for sort.Interface
func (rs RefList) Len() int { return len(rs) }

// Less returns true if i comes before j. Used for sort.Interface
func (rs RefList) Less(i, j int) bool { return rs[i].Peername+rs[i].Name < rs[j].Peername+rs[j].Name }

// Swap flips the positions of i and j. Used for sort.Interface
func (rs RefList) Swap(i, j int) { rs[i], rs[j] = rs[j], rs[i] }

// FlatbufferBytes turns refs into a byte slice of flatbuffer data
func FlatbufferBytes(rs RefList) []byte {
	builder := flatbuffers.NewBuilder(0)
	count := len(rs)

	offsets := make([]flatbuffers.UOffsetT, count)
	for i, l := range rs {
		offsets[i] = MarshalFlatbuffer(l, builder)
	}

	repofb.RepoStartRefsVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	refsvo := builder.EndVector(count)

	repofb.ReflistStart(builder)
	repofb.ReflistAddRefs(builder, refsvo)
	off := repofb.ReflistEnd(builder)

	builder.Finish(off)
	return builder.FinishedBytes()
}

// UnmarshalRefsFlatbuffer turns a repo flatbuffer into a list of refs
func UnmarshalRefsFlatbuffer(data []byte) (ls RefList, err error) {
	repoFb := repofb.GetRootAsReflist(data, 0)
	dec := &repofb.DatasetRef{}
	ls = make(RefList, repoFb.RefsLength())
	for i := 0; i < repoFb.RefsLength(); i++ {
		repoFb.Refs(dec, i)
		r, err := UnmarshalFlatbuffer(dec)
		if err != nil {
			return nil, err
		}
		ls[i] = r
	}

	return ls, nil
}

// MarshalFlatbuffer writes a ref to a builder
func MarshalFlatbuffer(r reporef.DatasetRef, builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	peername := builder.CreateString(r.Peername)
	profileID := builder.CreateString(r.ProfileID.String())
	name := builder.CreateString(r.Name)
	path := builder.CreateString(r.Path)
	fsiPath := builder.CreateString(r.FSIPath)

	repofb.DatasetRefStart(builder)
	repofb.DatasetRefAddPeername(builder, peername)
	repofb.DatasetRefAddProfileID(builder, profileID)
	repofb.DatasetRefAddName(builder, name)
	repofb.DatasetRefAddPath(builder, path)
	repofb.DatasetRefAddFsiPath(builder, fsiPath)
	repofb.DatasetRefAddPublished(builder, r.Published)
	return repofb.DatasetRefEnd(builder)
}

// UnmarshalFlatbuffer decodes a job from a flatbuffer
func UnmarshalFlatbuffer(rfb *repofb.DatasetRef) (r reporef.DatasetRef, err error) {
	r = reporef.DatasetRef{
		Peername:  string(rfb.Peername()),
		Name:      string(rfb.Name()),
		Path:      string(rfb.Path()),
		FSIPath:   string(rfb.FsiPath()),
		Published: rfb.Published(),
	}

	if pidstr := string(rfb.ProfileID()); pidstr != "" {
		r.ProfileID, err = profile.IDB58Decode(pidstr)
	}

	return r, err
}
