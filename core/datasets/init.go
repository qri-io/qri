package datasets

import (
	"io"
	"io/ioutil"

	"github.com/ipfs/go-datastore"
	ipfs "github.com/qri-io/castore/ipfs"
	"github.com/qri-io/dataset"
)

// AddFileResource adds a file from a given filepath on the local filesystem
func AddFileResource(ds *ipfs.Datastore, path string, rsc *dataset.Resource) (rkey datastore.Key, err error) {
	rkey = datastore.NewKey("")

	datahash, err := ds.AddAndPinPath(path)
	rsc.Path = datastore.NewKey("/ipfs/" + datahash)

	rdata, err := rsc.MarshalJSON()
	if err != nil {
		return
	}

	rhash, err := ds.AddAndPinBytes(rdata)
	if err != nil {
		return
	}

	rkey = datastore.NewKey("/ipfs/" + rhash)

	return
}

// AddReaderResource adds a resource from an io.Reader
// TODO - reverse the implementation, having AddBytesResource be a shorthand for AddReaderResource ;)
func AddReaderResource(ds *ipfs.Datastore, rdr io.Reader, rsc *dataset.Resource) (rkey datastore.Key, err error) {
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		return
	}
	return AddBytesResource(ds, data, rsc)
}

// AddBytesResource adds a slice of bytes as
func AddBytesResource(ds *ipfs.Datastore, data []byte, rsc *dataset.Resource) (rkey datastore.Key, err error) {
	rkey = datastore.NewKey("")

	datahash, err := ds.AddAndPinBytes(data)
	rsc.Path = datastore.NewKey("/ipfs/" + datahash)

	rdata, err := rsc.MarshalJSON()
	if err != nil {
		return
	}

	rhash, err := ds.AddAndPinBytes(rdata)
	if err != nil {
		return
	}

	rkey = datastore.NewKey("/ipfs/" + rhash)
	return
}
