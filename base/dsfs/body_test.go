package dsfs

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"
)

func TestLoadBody(t *testing.T) {
	ctx := context.Background()
	datasets, store, err := makeFilestore()
	if err != nil {
		t.Errorf("error creating test filestore: %s", err.Error())
		return
	}

	ds, err := LoadDataset(ctx, store, datasets["movies"])
	if err != nil {
		t.Errorf("error loading dataset: %s", err.Error())
		return
	}

	f, err := LoadBody(ctx, store, ds)
	if err != nil {
		t.Errorf("error loading data: %s", err.Error())
		return
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		t.Errorf("error reading data file: %s", err.Error())
		return
	}

	eq, err := ioutil.ReadFile("testdata/movies/body.csv")
	if err != nil {
		t.Errorf("error reading test file: %s", err.Error())
		return
	}

	if !bytes.Equal(data, eq) {
		t.Errorf("byte mismatch. expected: %s, got: %s", string(eq), string(data))
	}
}
