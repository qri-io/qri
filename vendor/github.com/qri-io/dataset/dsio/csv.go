package dsio

import (
	"encoding/csv"
	"github.com/qri-io/dataset"
	"io"
)

type CsvWriter struct {
	rowsWritten int
	w           *csv.Writer
	st          *dataset.Structure
}

func NewCsvWriter(st *dataset.Structure, w io.Writer) *CsvWriter {
	writer := csv.NewWriter(w)
	return &CsvWriter{
		st: st,
		w:  writer,
	}
}

func (w *CsvWriter) Structure() dataset.Structure {
	return *w.st
}

func (w *CsvWriter) WriteRow(data [][]byte) error {
	row := make([]string, len(data))
	for i, d := range data {
		row[i] = string(d)
	}
	return w.w.Write(row)
}

func (w *CsvWriter) Close() error {
	w.w.Flush()
	// no-op to satisfy interface
	return nil
}

type CsvReader struct {
	st         *dataset.Structure
	readHeader bool
	r          *csv.Reader
}

func NewCsvReader(st *dataset.Structure, r io.Reader) *CsvReader {
	return &CsvReader{
		st: st,
		r:  csv.NewReader(r),
	}
}

func (r *CsvReader) Structure() dataset.Structure {
	return *r.st
}

func (r *CsvReader) ReadRow() ([][]byte, error) {
	if !r.readHeader {
		if HasHeaderRow(r.st) {
			if _, err := r.r.Read(); err != nil {
				if err.Error() == "EOF" {
					return nil, nil
				}
				return nil, err
			}
		}
		r.readHeader = true
	}

	data, err := r.r.Read()
	if err != nil {
		return nil, err
	}
	row := make([][]byte, len(data))
	for i, d := range data {
		row[i] = []byte(d)
	}
	return row, nil
}

func HasHeaderRow(st *dataset.Structure) bool {
	if st.Format == dataset.CsvDataFormat && st.FormatConfig != nil {
		if csvOpt, ok := st.FormatConfig.(*dataset.CsvOptions); ok {
			return csvOpt.HeaderRow
		}
	}
	return false
}
