package cmd

import (
	"fmt"
	"github.com/ipfs/go-datastore"
	ipfs "github.com/qri-io/castore/ipfs"
	"github.com/qri-io/dataset"
)

func GetResource(store *ipfs.Datastore, key datastore.Key) (*dataset.Resource, error) {
	riface, err := store.Get(key)
	if err != nil {
		return nil, fmt.Errorf("error getting resource:", err.Error())
	}
	return dataset.UnmarshalResource(riface)
}

func GetStructuredData(store *ipfs.Datastore, key datastore.Key) ([]byte, error) {
	dataiface, err := store.Get(key)
	if err != nil {
		return nil, fmt.Errorf("error getting structured data for key: %s:", key.String(), err.Error())
	} else if databytes, ok := dataiface.([]byte); ok {
		return databytes, nil
	}
	return nil, fmt.Errorf("key: %s is not a slice of bytes", key.String())
}
