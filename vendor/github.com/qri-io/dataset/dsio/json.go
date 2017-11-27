package dsio

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
)

// TODO
type JsonReader struct {
	rowsRead    int
	initialized bool
	st          *dataset.Structure
	sc          *bufio.Scanner
}

func NewJsonReader(st *dataset.Structure, r io.Reader) *JsonReader {
	sc := bufio.NewScanner(r)
	jr := &JsonReader{
		st: st,
		sc: sc,
	}
	sc.Split(jr.scanJsonRow)
	return jr
}

func (r *JsonReader) Structure() dataset.Structure {
	return *r.st
}

func (r *JsonReader) ReadRow() ([][]byte, error) {
	more := r.sc.Scan()
	if !more {
		return nil, fmt.Errorf("EOF")
	}
	r.rowsRead++

	return [][]byte{r.sc.Bytes()}, r.sc.Err()
}

// initialIndex sets the scanner up to read data, advancing until the first
// entry in the top level array & setting the scanner split func to scan objects
func initialIndex(data []byte) (skip int, err error) {
	typ, err := datatypes.JsonArrayOrObject(data)
	if err != nil {
		// might not have initial closure, request more data
		return -1, err
	}
	if typ == "object" {
		return 0, fmt.Errorf("jsonReader top level must be an array")
	}

	// grab first opening bracked index to advance past
	// initial array closure
	idx := bytes.IndexByte(data, '[')
	return idx + 1, nil
}

func (r *JsonReader) scanJsonRow(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	depth := 0
	starti := -1
	stopi := -1

	if !r.initialized {
		skip, err := initialIndex(data)
		if err != nil {
			return 0, nil, err
		}
		if skip > 0 {
			r.initialized = true
			data = data[skip:]
		}
	}

LOOP:
	for i, b := range data {
		switch b {
		case '{', '[':
			if depth == 0 {
				starti = i
			}
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				stopi = i + 1
				break LOOP
			} else if depth < 0 {
				// if we encounter a closing bracket
				// before any depth, it's the end of the line
				return len(data), nil, nil
			}
		}
	}

	if stopi == -1 || starti == -1 {
		return 0, nil, nil
	}

	// return sliced data
	if starti < stopi {
		return stopi + 1, data[starti:stopi], nil
	}

	// Request more data.
	return 0, nil, nil
}

type JsonWriter struct {
	writeObjects bool
	rowsWritten  int
	st           *dataset.Structure
	wr           io.Writer
}

func NewJsonWriter(st *dataset.Structure, w io.Writer) *JsonWriter {
	writeObjects := true
	if opt, ok := st.FormatConfig.(*dataset.JsonOptions); ok {
		writeObjects = !opt.ArrayEntries
	}
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
			case datatypes.Json:
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
			case datatypes.Json:
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
	// if WriteRow is never called, write an empty array
	if w.rowsWritten == 0 {
		if _, err := w.wr.Write([]byte("[]")); err != nil {
			return fmt.Errorf("error writing initial `[`: %s", err.Error())
		}
		return nil
	}

	_, err := w.wr.Write([]byte{'\n', ']'})
	if err != nil {
		return fmt.Errorf("error closing writer: %s", err.Error())
	}
	return nil
}
