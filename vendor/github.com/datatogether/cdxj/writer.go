package cdxj

import (
	"bytes"
	"io"
	"sort"
)

// writer will write the following header:
var header = []byte("!OpenWayback-CDXJ 1.0\n")

// Writer writes to an io.Writer, create one with NewWriter
// You *must* call call Close to write the record to the
// specified writer
type Writer struct {
	writer  io.Writer
	records ByteRecords
}

// NewWriter allocates a new CDXJ Writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		writer:  w,
		records: make(ByteRecords, 0),
	}
}

// Write a record to the writer
func (w *Writer) Write(r *Record) error {
	data, err := r.MarshalCDXJ()
	if err != nil {
		return err
	}
	w.records = append(w.records, data)
	return nil
}

// Close dumps the writer to the underlying io.Writer
func (w *Writer) Close() error {
	sort.Sort(w.records)

	if _, err := w.writer.Write(header); err != nil {
		return err
	}

	for _, rec := range w.records {
		if _, err := w.writer.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

// ByteRecords implements sortable for a slice marshaled CDXJ byte slices
type ByteRecords [][]byte

func (a ByteRecords) Len() int           { return len(a) }
func (a ByteRecords) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByteRecords) Less(i, j int) bool { return bytes.Compare(a[i], a[j]) == -1 }
