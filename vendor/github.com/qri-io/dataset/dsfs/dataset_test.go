package dsfs

import (
	"encoding/json"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
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
	resource := dataset.NewDatasetRef(datastore.NewKey("resource1"))
	resource.Title = "now resource.Empty() == false"

	datapath, err := store.Put(memfs.NewMemfileBytes("data.csv", []byte("hello world")), false)
	if err != nil {
		t.Errorf("error putting test data in store: %s", err.Error())
		return
	}

	ds := &dataset.Dataset{
		Title: "test store",
		Structure: &dataset.Structure{
			Format: dataset.CsvDataFormat,
			Schema: &dataset.Schema{
				Fields: []*dataset.Field{},
			},
		},
		AbstractStructure: &dataset.Structure{
			Format: dataset.CsvDataFormat,
			Schema: &dataset.Schema{
				Fields: []*dataset.Field{},
			},
		},
		Query: &dataset.Query{
			Syntax: "dunno",
			Abstract: &dataset.AbstractQuery{
				Statement: "test statement",
			},
			Resources: map[string]*dataset.Dataset{
				"test": resource,
			},
		},
		AbstractQuery: &dataset.AbstractQuery{Statement: "select fooo from bar"},
		Data:          datapath,
	}

	key, err := SaveDataset(store, ds, true)
	if err != nil {
		t.Error(err.Error())
		return
	}

	hash := "/map/Qma9NTpbnpW4KR4LAFW8jypRRCWpJX84k2aX8Dc1yj2gQQ"
	if hash != key.String() {
		t.Errorf("key mismatch: %s != %s", hash, key.String())
		return
	}

	expectedEntries := 5
	if len(store.(memfs.MapStore)) != expectedEntries {
		t.Error("invalid number of entries added to store: %d != %d", expectedEntries, len(store.(memfs.MapStore)))
		return
	}

	f, err := store.Get(datastore.NewKey(hash))
	if err != nil {
		t.Errorf("error getting dataset file: %s", err.Error())
		return
	}

	result := &dataset.Dataset{}
	if err := json.NewDecoder(f).Decode(result); err != nil {
		t.Errorf("error decoding dataset json: %s", err.Error())
		return
	}

	if !result.AbstractQuery.IsEmpty() {
		t.Errorf("expected stored dataset.AbstractQuery to be a reference")
	}
	if !result.Query.IsEmpty() {
		t.Errorf("expected stored dataset.Query to be a reference")
	}
	if !result.Structure.IsEmpty() {
		t.Errorf("expected stored dataset.Structure to be a reference")
	}
	if !result.AbstractStructure.IsEmpty() {
		t.Errorf("expected stored dataset.AbstractStructure to be a reference")
	}

	qf, err := store.Get(result.Query.Path())
	if err != nil {
		t.Errorf("error getting query file: %s", err.Error())
		return
	}

	q := &dataset.Query{}
	if err := json.NewDecoder(qf).Decode(q); err != nil {
		t.Errorf("error decoding query json: %s", err.Error())
		return
	}

	if !q.Abstract.IsEmpty() {
		t.Errorf("expected stored query.Abstract to be a reference")
	}
	for name, ref := range q.Resources {
		if !ref.IsEmpty() {
			t.Errorf("expected stored query reference '%s' to be empty", name)
		}
	}
}
