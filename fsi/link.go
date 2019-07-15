package fsi

import (
	flatbuffers "github.com/google/flatbuffers/go"
	fsifb "github.com/qri-io/qri/fsi/fsi_fbs"
)

// links is a list of links
type links []*Link

// Remove deletes an entry from the list of links at an index
func (ls links) Remove(i int) links {
	ls[i] = ls[len(ls)-1]
	return ls[:len(ls)-1]
}

// FlatbufferBytes turns links into a byte slice of flatbuffer data
func (ls links) FlatbufferBytes() []byte {
	builder := flatbuffers.NewBuilder(0)
	count := len(ls)

	offsets := make([]flatbuffers.UOffsetT, count)
	for i, l := range ls {
		offsets[i] = l.MarshalFlatbuffer(builder)
	}

	fsifb.LinksStartListVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	linksvo := builder.EndVector(count)

	fsifb.LinksStart(builder)
	fsifb.LinksAddList(builder, linksvo)
	off := fsifb.LinksEnd(builder)

	builder.Finish(off)
	return builder.FinishedBytes()
}

func unmarshalLinksFlatbuffer(data []byte) (ls links, err error) {
	lsFb := fsifb.GetRootAsLinks(data, 0)
	dec := &fsifb.Link{}
	ls = make(links, lsFb.ListLength())
	for i := 0; i < lsFb.ListLength(); i++ {
		lsFb.List(dec, i)
		ls[i] = &Link{}
		if err := ls[i].UnmarshalFlatbuffer(dec); err != nil {
			return nil, err
		}
	}

	return ls, nil
}

// Link is a connection between a path and a dataset reference
type Link struct {
	Ref   string
	Path  string
	Alias string
}

// FlatbufferBytes formats a link as a flatbuffer byte slice
func (link *Link) FlatbufferBytes() []byte {
	builder := flatbuffers.NewBuilder(0)
	off := link.MarshalFlatbuffer(builder)
	builder.Finish(off)
	return builder.FinishedBytes()
}

// MarshalFlatbuffer writes a link to a builder
func (link *Link) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	ref := builder.CreateString(link.Ref)
	path := builder.CreateString(link.Path)
	alias := builder.CreateString(link.Alias)

	fsifb.LinkStart(builder)
	fsifb.LinkAddRef(builder, ref)
	fsifb.LinkAddPath(builder, path)
	fsifb.LinkAddAlias(builder, alias)
	return fsifb.LinkEnd(builder)
}

// UnmarshalFlatbuffer decodes a job from a flatbuffer
func (link *Link) UnmarshalFlatbuffer(l *fsifb.Link) error {
	*link = Link{
		Ref:   string(l.Ref()),
		Path:  string(l.Path()),
		Alias: string(l.Alias()),
	}
	return nil
}
