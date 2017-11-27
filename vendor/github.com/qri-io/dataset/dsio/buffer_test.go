package dsio

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/qri-io/dataset"
)

func TestBuffer(t *testing.T) {
	datasets, err := makeTestData()
	if err != nil {
		t.Errorf("error creating filestore", err.Error())
		return
	}

	ds := datasets["movies"].ds
	if err != nil {
		t.Errorf("error creating dataset: %s", err.Error())
		return
	}

	outst := &dataset.Structure{
		Format: dataset.JsonDataFormat,
		FormatConfig: &dataset.JsonOptions{
			ArrayEntries: false,
		},
		Schema: ds.Structure.Schema,
	}

	rbuf, err := NewBuffer(outst)
	if err != nil {
		t.Errorf("error allocating Buffer: %s", err.Error())
		return
	}

	rr, err := NewRowReader(ds.Structure, bytes.NewBuffer(datasets["movies"].data))
	if err != nil {
		t.Errorf("error allocating RowReader: %s", err.Error())
		return
	}

	if err = EachRow(rr, func(i int, row [][]byte, err error) error {
		if err != nil {
			return err
		}
		return rbuf.WriteRow(row)
	}); err != nil {
		t.Errorf("error writing rows: %s", err.Error())
		return
	}

	bst := rbuf.Structure()
	if err := dataset.CompareStructures(outst, &bst); err != nil {
		t.Errorf("buffer structure mismatch: %s", err.Error())
		return
	}

	if err := rbuf.Close(); err != nil {
		t.Errorf("error closing buffer: %s", err.Error())
		return
	}

	out := []interface{}{}
	if err := json.Unmarshal(rbuf.Bytes(), &out); err != nil {
		t.Errorf("error unmarshaling encoded bytes: %s", err.Error())
		return
	}

	if _, err = json.Marshal(out); err != nil {
		t.Errorf("error marshaling json data: %s", err.Error())
		return
	}

	// ioutil.WriteFile("testdata/movies_out.json", jsondata, 0777)
}
