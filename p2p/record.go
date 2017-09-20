package p2p

import (
	"github.com/ipfs/go-datastore"
	record "github.com/libp2p/go-libp2p-record"
	"github.com/qri-io/dataset"
)

// TODO - work in progress
func (qn *QriNode) PutQueryKey(key datastore.Key, q *dataset.Query) error {
	data, err := q.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = record.MakePutRecord(qn.privateKey, key.String(), data, true)
	if err != nil {
		return err
	}

	return nil
}
