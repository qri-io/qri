// dataset io defines writers & readers for datasets
package dsio

import (
	"io"

	"github.com/qri-io/dataset"
)

type RowWriter interface {
	Structure() dataset.Structure
	WriteRow(row [][]byte) error
	Close() error
}

type RowReader interface {
	Structure() dataset.Structure
	ReadRow() ([][]byte, error)
}

type RowReadWriter interface {
	Structure() dataset.Structure
	ReadRow() ([][]byte, error)
	WriteRow(row [][]byte) error
	Close() error
	Bytes() []byte
}

func NewRowWriter(st *dataset.Structure, w io.Writer) RowWriter {
	switch st.Format {
	case dataset.CsvDataFormat:
		return NewCsvWriter(st, w)
	case dataset.JsonDataFormat:
		return NewJsonWriter(st, w)
	case dataset.CdxjDataFormat:
		return NewCdxjWriter(st, w)
	default:
		// TODO - should this error or something?
		return nil
	}
}

func NewRowReader(st *dataset.Structure, r io.Reader) RowReader {
	switch st.Format {
	case dataset.CsvDataFormat:
		return NewCsvReader(st, r)
	case dataset.JsonDataFormat:
		// fmt.Errorf("json readers not yet supported")
		return nil
	case dataset.CdxjDataFormat:
		return NewCdxjReader(st, r)
	default:
		// fmt.Errorf("invalid format to create reader: %s", st.Format.String())
		return nil
	}
}
