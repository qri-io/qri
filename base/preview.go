package base

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
)

// MaxNumDatasetRowsInPreview is the highest number of rows a dataset preview
// can contain
const MaxNumDatasetRowsInPreview = 100

// CreatePreview generates a preview for a dataset version
func CreatePreview(ctx context.Context, r repo.Repo, ref dsref.Ref) (ds *dataset.Dataset, err error) {
	if ref.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	ds, err = dsfs.LoadDataset(ctx, r.Store(), ref.Path)
	if err != nil {
		log.Errorf("CreatePreview loading dataset: %s", err.Error())
		return nil, err
	}

	if err = ds.OpenBodyFile(ctx, r.Store()); err != nil {
		log.Errorf("CreatePreview opening body file: %s", err.Error())
		return nil, err
	}

	st := &dataset.Structure{
		Format: "json",
		Schema: ds.Structure.Schema,
	}

	data, err := ConvertBodyFile(ds.BodyFile(), ds.Structure, st, MaxNumDatasetRowsInPreview, 0, false)
	if err != nil {
		log.Errorf("CreatePreview converting body file: %s", err.Error())
		return nil, err
	}

	ds.Peername = ref.Username
	ds.Name = ref.Name
	ds.Path = ref.Path
	ds.Body = json.RawMessage(data)
	return ds, nil
}
