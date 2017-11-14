package repo

// import (
// 	"encoding/json"
// 	"fmt"
// 	"testing"

// 	"github.com/ipfs/go-datastore"
// 	"github.com/qri-io/cafs/memfs"
// 	"github.com/qri-io/dataset"
// 	"github.com/qri-io/qri/repo/profile"
// )

// var (
// 	ds1 = &dataset.Dataset{
// 		Previous: datastore.NewKey(""),
// 	}
// 	ds2 = &dataset.Dataset{
// 		Previous: datastore.NewKey(""),
// 	}
// )

// func TestRepoGraph(t *testing.T) {
// 	store := memfs.NewMapstore()
// 	p := &profile.Profile{}

// 	r, err := NewMemRepo(p, store, nil, nil)
// 	if err != nil {
// 		t.Errorf("error creating test repo: %s", err.Error())
// 		return
// 	}

// 	data1p, _ := store.Put(memfs.NewMemfileBytes("data1", []byte("dataset_1")), true)
// 	ds1.Data = data1p
// 	ds1j, _ := ds1.MarshalJSON()
// 	ds1p, err := store.Put(memfs.NewMemfileBytes("ds1", ds1j), true)
// 	if err != nil {
// 		t.Errorf("error putting dataset: %s", err.Error())
// 		return
// 	}
// 	r.PutDataset(ds1p, ds1)
// 	r.PutName("ds1", ds1p)

// 	data2p, _ := store.Put(memfs.NewMemfileBytes("data1", []byte("dataset_2")), true)
// 	ds2.Data = data2p
// 	ds2j, _ := ds1.MarshalJSON()
// 	ds2p, err := store.Put(memfs.NewMemfileBytes("ds2", ds2j), true)
// 	if err != nil {
// 		t.Errorf("error putting dataset: %s", err.Error())
// 		return
// 	}
// 	r.PutDataset(ds2p, ds2)
// 	r.PutName("ds1", ds2p)

// 	node, err := RepoGraph(r)
// 	if err != nil {
// 		t.Errorf("error generating repo graph: %s", err.Error())
// 		return
// 	}

// 	data, err := json.Marshal(node)
// 	if err != nil {
// 		t.Errorf("json marshal error: %s", err.Error())
// 		return
// 	}
// 	fmt.Println(data)
// }
