package repo

import (
	flatbuffers "github.com/google/flatbuffers/go"
	repofb "github.com/qri-io/qri/repo/repo_fbs"
)

// Refs is a list of refs
type Refs []DatasetRef

// Len returns the length of refs
func (rs Refs) Len() int { return len(rs) }

// Less returns true if i comes before j
func (rs Refs) Less(i, j int) bool { return rs[i].Peername+rs[i].Name < rs[j].Peername+rs[j].Name }

// Swap flips the positions of i and j
func (rs Refs) Swap(i, j int) { rs[i], rs[j] = rs[j], rs[i] }

// Remove deletes an entry from the list of refs at an index
func (rs Refs) Remove(i int) Refs {
	rs[i] = rs[len(rs)-1]
	return rs[:len(rs)-1]
}

// FlatbufferBytes turns refs into a byte slice of flatbuffer data
func (rs Refs) FlatbufferBytes() []byte {
	builder := flatbuffers.NewBuilder(0)
	count := len(rs)

	offsets := make([]flatbuffers.UOffsetT, count)
	for i, l := range rs {
		offsets[i] = l.MarshalFlatbuffer(builder)
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
func UnmarshalRefsFlatbuffer(data []byte) (ls Refs, err error) {
	repoFb := repofb.GetRootAsReflist(data, 0)
	dec := &repofb.DatasetRef{}
	ls = make(Refs, repoFb.RefsLength())
	for i := 0; i < repoFb.RefsLength(); i++ {
		repoFb.Refs(dec, i)
		ls[i] = DatasetRef{}
		if err := ls[i].UnmarshalFlatbuffer(dec); err != nil {
			return nil, err
		}
	}

	return ls, nil
}
