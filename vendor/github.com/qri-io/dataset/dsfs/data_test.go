package dsfs

import (
	"bytes"
	"testing"
)

func TestLoadRows(t *testing.T) {
	datasets, store, err := makeFilestore()
	if err != nil {
		t.Errorf("error creating test filestore: %s", err.Error())
		return
	}

	ds, err := LoadDataset(store, datasets["cities"])
	if err != nil {
		t.Errorf("error loading dataset: %s", err.Error())
		return
	}

	data, err := LoadRows(store, ds, 2, 2)
	if err != nil {
		t.Errorf("raw data row error: %s", err.Error())
		return
	}

	expect := []byte(`chicago,300000,44.4,true
chatham,35000,65.25,true
`)

	if !bytes.Equal(expect, data) {
		t.Errorf("data mismatch. expected: %s, got: %s", string(expect), string(data))
		return
	}
}
