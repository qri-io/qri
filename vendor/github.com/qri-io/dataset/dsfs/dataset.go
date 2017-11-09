package dsfs

import (
	"encoding/json"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
)

// Load a dataset from a cafs
func LoadDataset(store cafs.Filestore, path datastore.Key) (*dataset.Dataset, error) {
	ds := &dataset.Dataset{}

	data, err := fileBytes(store.Get(path))
	// if err != nil {
	// 	return nil, fmt.Errorf("error getting file bytes: %s", err.Error())
	// }

	// TODO - for some reason files are sometimes coming back empty from IPFS,
	// every now & then. In the meantime, let's give a second try if data is empty
	if err != nil || len(data) == 0 {
		data, err = fileBytes(store.Get(path))
		if err != nil {
			return nil, fmt.Errorf("error getting file bytes: %s", err.Error())
		}
	}

	ds, err = dataset.UnmarshalDataset(data)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling %s file: %s", PackageFileDataset.String(), err.Error())
	}

	if err := DerefDatasetStructure(store, ds); err != nil {
		return nil, fmt.Errorf("error dereferencing %s file: %s", PackageFileStructure, err.Error())
	}

	if err := DerefDatasetQuery(store, ds); err != nil {
		return nil, fmt.Errorf("error dereferencing %s file: %s", PackageFileQuery, err.Error())
	}

	return ds, nil
}

// DerefDatasetStructure derferences a dataset's structure element if required
// should be a no-op if ds.Structure is nil or isn't a reference
func DerefDatasetStructure(store cafs.Filestore, ds *dataset.Dataset) error {
	if ds.Structure != nil && ds.Structure.IsEmpty() && ds.Structure.Path().String() != "" {
		st, err := LoadStructure(store, ds.Structure.Path())
		if err != nil {
			return fmt.Errorf("error loading dataset structure: %s", err.Error())
		}
		ds.Structure = st
	}
	return nil
}

// DerefDatasetQuery derferences a dataset's query element if required
// should be a no-op if ds.Structure is nil or isn't a reference
func DerefDatasetQuery(store cafs.Filestore, ds *dataset.Dataset) error {
	if ds.Query != nil && ds.Query.IsEmpty() && ds.Query.Path().String() != "" {
		q, err := LoadQuery(store, ds.Query.Path())
		if err != nil {
			return fmt.Errorf("error loading dataset query: %s", err.Error())
		}
		ds.Query = q
	}
	return nil
}

func SaveDataset(store cafs.Filestore, ds *dataset.Dataset, pin bool) (datastore.Key, error) {
	if ds == nil {
		return datastore.NewKey(""), nil
	}

	fileTasks := 0
	addedDataset := false
	adder, err := store.NewAdder(pin, true)
	if err != nil {
		return datastore.NewKey(""), fmt.Errorf("error creating new adder: %s", err.Error())
	}

	// if dataset contains no references, place directly in.
	// TODO - this might not constitute a valid dataset. should we be
	// validating datasets in here?
	if ds.Query == nil && ds.Structure == nil && ds.Resources == nil {
		fileTasks++
		dsdata, err := json.Marshal(ds)
		if err != nil {
			return datastore.NewKey(""), fmt.Errorf("error marshaling dataset to json: %s", err.Error())
		}
		adder.AddFile(memfs.NewMemfileBytes(PackageFileDataset.String(), dsdata))
		addedDataset = true
	}

	if ds.Query != nil {
		fileTasks++
		qdata, err := json.Marshal(ds.Query)
		if err != nil {
			return datastore.NewKey(""), fmt.Errorf("error marshaling dataset query to json: %s", err.Error())
		}
		adder.AddFile(memfs.NewMemfileBytes(PackageFileQuery.String(), qdata))
	}

	if ds.Structure != nil {
		fileTasks++
		stdata, err := json.Marshal(ds.Structure)
		if err != nil {
			return datastore.NewKey(""), fmt.Errorf("error marshaling dataset structure to json: %s", err.Error())
		}
		adder.AddFile(memfs.NewMemfileBytes(PackageFileStructure.String(), stdata))

		fileTasks++
		asdata, err := json.Marshal(ds.Structure.Abstract())
		if err != nil {
			return datastore.NewKey(""), fmt.Errorf("error marshaling dataset abstract structure to json: %s", err.Error())
		}
		adder.AddFile(memfs.NewMemfileBytes(PackageFileAbstractStructure.String(), asdata))

		fileTasks++
		data, err := store.Get(ds.Data)
		if err != nil {
			return datastore.NewKey(""), fmt.Errorf("error getting dataset raw data: %s", err.Error())
		}
		adder.AddFile(memfs.NewMemfileReader("data."+ds.Structure.Format.String(), data))
	}

	// if ds.Previous != nil {
	// }

	// for name, d := range ds.Resources {
	//  if d.path.String() != "" && d.IsEmpty() {
	//    continue
	//  } else if d != nil {
	//    // dspath, err := d.Save(store, pin)
	//    // if err != nil {
	//    //  return datastore.NewKey(""), fmt.Errorf("error saving dataset resource: %s", err.Error())
	//    // }
	//    // ds.Resources[name] = &Dataset{path: dspath}
	//  }
	// }

	var path datastore.Key
	done := make(chan error, 0)
	go func() {
		for ao := range adder.Added() {
			path = ao.Path
			switch ao.Name {
			case PackageFileStructure.String():
				ds.Structure = dataset.NewStructureRef(ao.Path)
			case PackageFileAbstractStructure.String():
				ds.AbstractStructure = dataset.NewStructureRef(ao.Path)
			case PackageFileQuery.String():
				ds.Query = dataset.NewQueryRef(ao.Path)
			case "resources":
			}

			fileTasks--
			if fileTasks == 0 {
				if !addedDataset {
					dsdata, err := json.Marshal(ds)
					if err != nil {
						done <- err
						return
					}

					adder.AddFile(memfs.NewMemfileBytes(PackageFileDataset.String(), dsdata))
				}
				//
				if err := adder.Close(); err != nil {
					done <- err
					return
				}
			}
		}
		done <- nil
	}()

	err = <-done

	// ok, this is a horrible hack to deal with the fact that the location of
	// the actual dataset.json on ipfs will be /[hash]/dataset.json, a property
	// that may or may not apply to other cafs implementations.
	// We want to store the reference to the directory hash, so the
	// /[hash]/dataset.json form is desirable, becuase we can do stuff like
	// /[hash]/abstract_structure.json, and so on, but it's hard to extract
	// in a clean way. maybe a function that re-extracts this info on either
	// the cafs interface, or the concrete cafs/ipfs implementation?
	// TODO - remove this in favour of some sort of tree-walking
	if _, ok := store.(*ipfs_filestore.Filestore); ok {
		path = datastore.NewKey(path.String() + "/" + PackageFileDataset.String())
	}
	return path, err
}
