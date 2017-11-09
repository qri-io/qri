package dsio

import (
	"bytes"
	"fmt"
	"testing"
)

func TestEachRow(t *testing.T) {
	datasets, err := makeTestData()
	if err != nil {
		t.Errorf("error creating test filestore: %s", err.Error())
		return
	}

	ds := datasets["cities"].ds
	expect := datasets["cities"].rows
	rr := NewRowReader(ds.Structure, bytes.NewBuffer(datasets["cities"].data))
	err = EachRow(rr, func(i int, data [][]byte, err error) error {
		if err != nil {
			return err
		}

		if len(expect[i]) != len(data) {
			return fmt.Errorf("data length mismatch. expected %d, got: %d", len(expect[i]), len(data))
		}

		for j, cell := range data {
			if !bytes.Equal(expect[i][j], cell) {
				return fmt.Errorf("result mismatch. row: %d, cell: %d. %s != %s", i, j, string(expect[i][j]), string(cell))
			}
		}

		return nil
	})
	if err != nil {
		t.Errorf("eachrow error: %s", err.Error())
		return
	}
}
