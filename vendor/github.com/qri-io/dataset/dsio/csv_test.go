package dsio

import (
	"bytes"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
	"testing"
)

const data = `a,b,c,d
a,b,c,d
a,b,c,d
a,b,c,d
a,b,c,d
`

func TestCsvReader(t *testing.T) {
	st := &dataset.Structure{
		Format: dataset.CsvDataFormat,
		Schema: &dataset.Schema{
			Fields: []*dataset.Field{
				&dataset.Field{Name: "a", Type: datatypes.String},
				&dataset.Field{Name: "b", Type: datatypes.String},
				&dataset.Field{Name: "c", Type: datatypes.String},
				&dataset.Field{Name: "d", Type: datatypes.String},
			},
		},
	}

	buf := bytes.NewBuffer([]byte(data))
	rdr := NewRowReader(st, buf)
	count := 0
	for {
		row, err := rdr.ReadRow()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Errorf("unexpected error: %s", err.Error())
			return
		}

		if len(row) != 4 {
			t.Errorf("invalid row length for row %d. expected %d, got %d", count, 4, len(row))
		}

		count++
	}
	if count != 5 {
		t.Errorf("expected: %d rows, got: %d", 5, count)
	}
}
