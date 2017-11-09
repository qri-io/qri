package dsfs

import (
	"encoding/json"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
)

// LoadQuery loads a query from a given path in a store
func LoadQuery(store cafs.Filestore, path datastore.Key) (q *dataset.Query, err error) {
	data, err := fileBytes(store.Get(path))
	if err != nil {
		return nil, fmt.Errorf("error loading query raw data: %s", err.Error())
	}

	return dataset.UnmarshalQuery(data)
}

func SaveQuery(store cafs.Filestore, q *dataset.Query, pin bool) (datastore.Key, error) {
	if q == nil {
		return datastore.NewKey(""), nil
	}

	// *don't* need to break query out into different structs.
	// stpath, err := q.Structure.Save(store)
	// if err != nil {
	//  return datastore.NewKey(""), err
	// }

	qdata, err := json.Marshal(q)
	if err != nil {
		return datastore.NewKey(""), fmt.Errorf("error marshaling query data to json: %s", err.Error())
	}

	return store.Put(memfs.NewMemfileBytes("query.json", qdata), pin)
}
