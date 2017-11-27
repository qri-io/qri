// dataset io defines writers & readers for datasets
package dsio

import (
	"fmt"
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

// NewRowReader allocates a RowReader based on a given structure
func NewRowReader(st *dataset.Structure, r io.Reader) (RowReader, error) {
	switch st.Format {
	case dataset.CsvDataFormat:
		return NewCsvReader(st, r), nil
	case dataset.JsonDataFormat:
		return NewJsonReader(st, r), nil
	case dataset.CdxjDataFormat:
		return NewCdxjReader(st, r), nil
	case dataset.UnknownDataFormat:
		return nil, fmt.Errorf("structure must have a data format")
	default:
		return nil, fmt.Errorf("invalid format to create reader: %s", st.Format.String())
	}
}

// NewRowWriter allocates a RowWriter based on a given structure
func NewRowWriter(st *dataset.Structure, w io.Writer) (RowWriter, error) {
	switch st.Format {
	case dataset.CsvDataFormat:
		return NewCsvWriter(st, w), nil
	case dataset.JsonDataFormat:
		return NewJsonWriter(st, w), nil
	case dataset.CdxjDataFormat:
		return NewCdxjWriter(st, w), nil
	case dataset.UnknownDataFormat:
		return nil, fmt.Errorf("structure must have a data format")
	default:
		return nil, fmt.Errorf("invalid format to create writer: %s", st.Format.String())
	}
}
