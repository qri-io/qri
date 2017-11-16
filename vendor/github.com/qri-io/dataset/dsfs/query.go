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

func SaveQuery(store cafs.Filestore, q *dataset.Query, pin bool) (path datastore.Key, err error) {
	// copy query
	save := &dataset.Query{}
	save.Assign(q)

	absp, err := SaveAbstractQuery(store, save.Abstract, pin)
	if err != nil {
		return datastore.NewKey(""), fmt.Errorf("error saving abstract query: %s", err.Error())
	}
	save.Abstract = dataset.NewAbstractQueryRef(absp)

	// convert any full datasets to path references
	for name, d := range save.Resources {
		if d.Path().String() != "" && d.IsEmpty() {
			continue
		} else if d != nil {
			save.Resources[name] = dataset.NewDatasetRef(d.Path())
		}
	}

	qdata, err := save.MarshalJSON()
	if err != nil {
		return datastore.NewKey(""), fmt.Errorf("error marshaling query data to json: %s", err.Error())
	}

	return store.Put(memfs.NewMemfileBytes(PackageFileQuery.String(), qdata), pin)
}

func queryFile(q *dataset.Query) (cafs.File, error) {
	if q == nil {
		return nil, nil
	}
	if !q.Abstract.IsEmpty() {
		return nil, fmt.Errorf("query abstract query must be a reference to generate a query file")
	}

	// convert any full datasets to path references
	for name, d := range q.Resources {
		if d.Path().String() != "" && d.IsEmpty() {
			continue
		} else if d != nil {
			q.Resources[name] = dataset.NewDatasetRef(d.Path())
		}
	}

	qdata, err := json.Marshal(q)
	if err != nil {
		return nil, fmt.Errorf("error marshaling query data to json: %s", err.Error())
	}

	return memfs.NewMemfileBytes(PackageFileQuery.String(), qdata), nil
}

// LoadAbstractQuery loads a query from a given path in a store
func LoadAbstractQuery(store cafs.Filestore, path datastore.Key) (q *dataset.AbstractQuery, err error) {
	data, err := fileBytes(store.Get(path))
	if err != nil {
		return nil, fmt.Errorf("error loading query raw data: %s", err.Error())
	}

	return dataset.UnmarshalAbstractQuery(data)
}

func SaveAbstractQuery(store cafs.Filestore, q *dataset.AbstractQuery, pin bool) (datastore.Key, error) {
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
