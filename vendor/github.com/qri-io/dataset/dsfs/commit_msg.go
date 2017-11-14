package dsfs

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
)

// LoadCommitMsg loads a commit from a given path in a store
func LoadCommitMsg(store cafs.Filestore, path datastore.Key) (st *dataset.CommitMsg, err error) {
	data, err := fileBytes(store.Get(path))
	if err != nil {
		return nil, fmt.Errorf("error loading commit file: %s", err.Error())
	}
	return dataset.UnmarshalCommitMsg(data)
}

func SaveCommitMsg(store cafs.Filestore, s *dataset.CommitMsg, pin bool) (path datastore.Key, err error) {
	file, err := jsonFile(PackageFileCommitMsg.String(), s)
	if err != nil {
		return datastore.NewKey(""), fmt.Errorf("error saving json commit file: %s", err.Error())
	}
	return store.Put(file, pin)
}
