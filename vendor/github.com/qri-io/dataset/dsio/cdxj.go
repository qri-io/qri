package dsio

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/datatogether/cdxj"
	"github.com/qri-io/dataset"
)

type CdxjWriter struct {
	rowsWritten int
	st          *dataset.Structure
	w           *cdxj.Writer
}

func NewCdxjWriter(st *dataset.Structure, w io.Writer) *CdxjWriter {
	writer := cdxj.NewWriter(w)
	return &CdxjWriter{
		st: st,
		w:  writer,
	}
}

func (w *CdxjWriter) Structure() dataset.Structure {
	return *w.st
}

func (w *CdxjWriter) WriteRow(data [][]byte) error {
	r := &cdxj.Record{}
	if err := r.UnmarshalCDXJ(bytes.Join(data, []byte(" "))); err != nil {
		return err
	}
	return w.WriteRecord(r)
}

func (w *CdxjWriter) WriteRecord(rec *cdxj.Record) error {
	return w.w.Write(rec)
}

func (w *CdxjWriter) Close() error {
	return w.w.Close()
}

type CdxjReader struct {
	st *dataset.Structure
	r  *cdxj.Reader
}

func NewCdxjReader(st *dataset.Structure, r io.Reader) *CdxjReader {
	return &CdxjReader{
		st: st,
		r:  cdxj.NewReader(r),
	}
}

func (r *CdxjReader) Structure() dataset.Structure {
	return *r.st
}

func (r *CdxjReader) ReadRow() ([][]byte, error) {
	rec, err := r.r.Read()
	if err != nil {
		return nil, err
	}

	row := make([][]byte, 4)
	row[0] = []byte(rec.Uri)
	row[1] = []byte(rec.Timestamp.String())
	row[2] = []byte(rec.RecordType.String())
	row[3], err = json.Marshal(rec.JSON)
	if err != nil {
		return nil, err
	}
	return row, nil
}
