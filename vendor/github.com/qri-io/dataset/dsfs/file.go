package dsfs

import (
	"encoding/json"
	"io/ioutil"

	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfs"
)

func fileBytes(file cafs.File, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(file)
}

func jsonFile(name string, m json.Marshaler) (cafs.File, error) {
	data, err := m.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return memfs.NewMemfileBytes(name, data), nil
}
