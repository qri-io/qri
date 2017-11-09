package dsio

import (
	"fmt"
	"io"
	"strconv"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
)

type JsonWriter struct {
	writeObjects bool
	rowsWritten  int
	st           *dataset.Structure
	wr           io.Writer
}

func NewJsonWriter(st *dataset.Structure, w io.Writer, writeObjects bool) *JsonWriter {
	return &JsonWriter{
		writeObjects: writeObjects,
		st:           st,
		wr:           w,
	}
}

func (w *JsonWriter) Structure() dataset.Structure {
	return *w.st
}

func (w *JsonWriter) WriteRow(row [][]byte) error {
	if w.rowsWritten == 0 {
		if _, err := w.wr.Write([]byte{'['}); err != nil {
			return fmt.Errorf("error writing initial `[`: %s", err.Error())
		}
	}

	if w.writeObjects {
		return w.writeObjectRow(row)
	}
	return w.writeArrayRow(row)
}

func (w *JsonWriter) writeObjectRow(row [][]byte) error {
	enc := []byte{',', '\n', '{'}
	if w.rowsWritten == 0 {
		enc = enc[1:]
	}
	for i, c := range row {
		f := w.st.Schema.Fields[i]
		ent := []byte(",\"" + f.Name + "\":")
		if i == 0 {
			ent = ent[1:]
		}
		if c == nil || len(c) == 0 {
			ent = append(ent, []byte("null")...)
		} else {
			switch f.Type {
			case datatypes.String:
				ent = append(ent, []byte(strconv.Quote(string(c)))...)
			case datatypes.Float, datatypes.Integer:
				// if len(c) == 0 {
				// 	ent = append(ent, []byte("null")...)
				// } else {
				// 	ent = append(ent, c...)
				// }
				ent = append(ent, c...)
			case datatypes.Boolean:
				// TODO - coerce to true & false specifically
				ent = append(ent, c...)
			default:
				ent = append(ent, []byte(strconv.Quote(string(c)))...)
			}
		}

		enc = append(enc, ent...)
	}

	enc = append(enc, '}')
	if _, err := w.wr.Write(enc); err != nil {
		return fmt.Errorf("error writing json object row to writer: %s", err.Error())
	}

	w.rowsWritten++
	return nil
}

func (w *JsonWriter) writeArrayRow(row [][]byte) error {
	enc := []byte{',', '\n', '['}
	if w.rowsWritten == 0 {
		enc = enc[1:]
	}
	for i, c := range row {
		f := w.st.Schema.Fields[i]
		ent := []byte(",")
		if i == 0 {
			ent = ent[1:]
		}
		if c == nil || len(c) == 0 {
			ent = append(ent, []byte("null")...)
		} else {
			switch f.Type {
			case datatypes.String:
				ent = append(ent, []byte(strconv.Quote(string(c)))...)
			case datatypes.Float, datatypes.Integer:
				// TODO - decide on weather or not to supply default values
				// if len(c) == 0 {
				// ent = append(ent, []byte("0")...)
				// } else {
				ent = append(ent, c...)
				// }
			case datatypes.Boolean:
				// TODO - coerce to true & false specifically
				// if len(c) == 0 {
				// ent = append(ent, []byte("false")...)
				// }
				ent = append(ent, c...)
			default:
				ent = append(ent, []byte(strconv.Quote(string(c)))...)
			}
		}

		enc = append(enc, ent...)
	}

	enc = append(enc, ']')
	if _, err := w.wr.Write(enc); err != nil {
		return fmt.Errorf("error writing closing `]`: %s", err.Error())
	}

	w.rowsWritten++
	return nil
}

func (w *JsonWriter) Close() error {
	_, err := w.wr.Write([]byte{'\n', ']'})
	if err != nil {
		return fmt.Errorf("error closing writer: %s", err.Error())
	}
	return nil
}

// TODO
type JsonReader struct {
}
