package dsfs

import (
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"testing"
)

func TestLoadDataset(t *testing.T) {
	store := memfs.NewMapstore()
	apath, err := SaveDataset(store, AirportCodes, true)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	_, err = LoadDataset(store, apath)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestDatasetSave(t *testing.T) {
	store := memfs.NewMapstore()

	ds := &dataset.Dataset{
		Title: "test store",
		Query: &dataset.Query{
			Syntax:    "dunno",
			Statement: "test statement",
		},
	}

	key, err := SaveDataset(store, ds, true)
	if err != nil {
		t.Error(err.Error())
		return
	}

	hash := "/map/Qmc1e6ytPKJQ7YWNnms8GY7DEei8FXkbymbeseqQMD8nZz"
	if hash != key.String() {
		t.Errorf("key mismatch: %s != %s", hash, key.String())
		return
	}

	if len(store.(memfs.MapStore)) != 2 {
		t.Error("invalid number of entries added to store: %d != %d", 2, len(store.(memfs.MapStore)))
		return
	}
	// fmt.Println(string(store.(memfs.MapStore)[datastore.NewKey("/mem/Qmdv5WeDGw1f6pw4DSYQdsugNDFUqHw9FuFU8Gu7T4PUqF")].([]byte)))
}
