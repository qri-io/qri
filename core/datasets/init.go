package datasets

import (
	"io"
	"io/ioutil"

	"github.com/ipfs/go-datastore"
	ipfs "github.com/qri-io/castore/ipfs"
	"github.com/qri-io/dataset"
)

// AddFileStructure adds a file from a given filepath on the local filesystem
func AddFileStructure(ds *ipfs.Datastore, path string, st *dataset.Structure) (structkey, datakey datastore.Key, err error) {
	structkey = datastore.NewKey("")

	datahash, err := ds.AddAndPinPath(path)
	datakey = datastore.NewKey("/ipfs/" + datahash)

	stdata, err := st.MarshalJSON()
	if err != nil {
		return
	}

	sthash, err := ds.AddAndPinBytes(stdata)
	if err != nil {
		return
	}

	structkey = datastore.NewKey("/ipfs/" + sthash)

	return
}

// AddReaderStructure adds a resource from an io.Reader
// TODO - reverse the implementation, having AddBytesStructure be a shorthand for AddReaderStructure ;)
func AddReaderStructure(ds *ipfs.Datastore, rdr io.Reader, rsc *dataset.Structure) (stkey, datakey datastore.Key, err error) {
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		return
	}
	return AddBytesStructure(ds, data, rsc)
}

// AddBytesStructure adds a slice of bytes to ipfs
func AddBytesStructure(ds *ipfs.Datastore, data []byte, st *dataset.Structure) (stkey, datakey datastore.Key, err error) {
	stkey = datastore.NewKey("")

	datahash, err := ds.AddAndPinBytes(data)
	datakey = datastore.NewKey("/ipfs/" + datahash)

	stdata, err := st.MarshalJSON()
	if err != nil {
		return
	}

	sthash, err := ds.AddAndPinBytes(stdata)
	if err != nil {
		return
	}

	stkey = datastore.NewKey("/ipfs/" + sthash)
	return
}
