package dsio

import (
	"bytes"
	"github.com/qri-io/dataset"
)

func NewBuffer(st *dataset.Structure) (*Buffer, error) {
	buf := &bytes.Buffer{}
	r, err := NewRowReader(st, buf)
	if err != nil {
		return nil, err
	}
	w, err := NewRowWriter(st, buf)
	if err != nil {
		return nil, err
	}

	return &Buffer{
		structure: st,
		r:         r,
		w:         w,
		buf:       buf,
	}, nil
}

type Buffer struct {
	structure *dataset.Structure
	r         RowReader
	w         RowWriter
	buf       *bytes.Buffer
}

func (b *Buffer) Structure() dataset.Structure {
	return *b.structure
}

func (b *Buffer) ReadRow() ([][]byte, error) {
	return b.r.ReadRow()
}

func (b *Buffer) WriteRow(row [][]byte) error {
	return b.w.WriteRow(row)
}

func (b *Buffer) Close() error {
	return b.w.Close()
}

func (b *Buffer) Bytes() []byte {
	return b.buf.Bytes()
}
