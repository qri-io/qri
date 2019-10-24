package dsfs

import (
	"encoding/json"
	"io/ioutil"

	"github.com/qri-io/qfs"
)

// JSONFile is a convenenience method for creating a file from a json.Marshaller
func JSONFile(name string, m json.Marshaler) (qfs.File, error) {
	data, err := m.MarshalJSON()
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}
	return qfs.NewMemfileBytes(name, data), nil
}

func fileBytes(file qfs.File, err error) ([]byte, error) {
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}
	return ioutil.ReadAll(file)
}
