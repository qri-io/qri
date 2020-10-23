package dsfs

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
)

// ErrStrictMode indicates a dataset failed validation when it is required to
// pass (Structure.Strict == true)
var ErrStrictMode = fmt.Errorf("dataset body did not validate against schema in strict-mode")

// BaseTabularSchema is the base schema for tabular data
// NOTE: Do not use if possible, prefer github.com/qri-io/dataset/tabular
// TODO(dustmop): Possibly move this to tabular package
var BaseTabularSchema = map[string]interface{}{
	"type": "array",
	"items": map[string]interface{}{
		"type":  "array",
		"items": []interface{}{},
	},
}

// loadStructure assumes path is valid
func loadStructure(ctx context.Context, fs qfs.Filesystem, path string) (st *dataset.Structure, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading structure file: %s", err.Error())
	}
	return dataset.UnmarshalStructure(data)
}
