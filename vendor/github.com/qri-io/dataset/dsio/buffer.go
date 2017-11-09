package dsio

import (
	"bytes"
	"github.com/qri-io/dataset"
)

func NewBuffer(st *dataset.Structure) *Buffer {
	buf := &bytes.Buffer{}
	return &Buffer{
		structure: st,
		r:         NewRowReader(st, buf),
		w:         NewRowWriter(st, buf),
		buf:       buf,
	}
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
