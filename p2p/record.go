package p2p

import (
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"

	record "gx/ipfs/QmbxkgUceEcuSZ4ZdBA3x74VUDSSYjHYmmeEqkjxbtZ6Jg/go-libp2p-record"
)

// TODO - work in progress
func (qn *QriNode) PutQueryKey(key datastore.Key, q *dataset.Transform) error {
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
