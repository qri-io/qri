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
			ObjectEntries: true,
		},
		Schema: ds.Structure.Schema,
	}

	buf := NewBuffer(outst)

	rr := NewRowReader(ds.Structure, bytes.NewBuffer(datasets["movies"].data))
	err = EachRow(rr, func(i int, row [][]byte, err error) error {
		if err != nil {
			return err
		}
		return buf.WriteRow(row)
	})

	if err != nil {
		t.Errorf("error iterating through rows: %s", err.Error())
		return
	}

	if err := buf.Close(); err != nil {
		t.Errorf("error closing buffer: %s", err.Error())
		return
	}

	out := []interface{}{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Errorf("error unmarshaling encoded bytes: %s", err.Error())
		return
	}

	if _, err = json.Marshal(out); err != nil {
		t.Errorf("error marshaling json data: %s", err.Error())
		return
	}

	// ioutil.WriteFile("testdata/movies_out.json", jsondata, 0777)
}
