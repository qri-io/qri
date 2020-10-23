package base

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
)

const (
	// MaxNumDatasetRowsInPreview is the highest number of rows a dataset preview
	// can contain
	MaxNumDatasetRowsInPreview = 100
	// MaxReadmePreviewBytes determines the maximum amount of bytes a readme
	// preview can be. three bytes less than 1000 to make room for an elipsis
	MaxReadmePreviewBytes = 997
)

// CreatePreview generates a preview for a dataset version
func CreatePreview(ctx context.Context, fs qfs.Filesystem, ref dsref.Ref) (ds *dataset.Dataset, err error) {
	if ref.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	ds, err = dsfs.LoadDataset(ctx, fs, ref.Path)
	if err != nil {
		log.Errorf("CreatePreview loading dataset: %s", err.Error())
		return nil, err
	}

	if ds.Readme != nil {
		if err := openReadme(ctx, fs, ds); err != nil {
			log.Errorf("OpeningReadme: %s", err.Error())
			return nil, err
		}

		if readmeFile := ds.Readme.ScriptFile(); readmeFile != nil {
			ds.Readme.ScriptBytes, err = ioutil.ReadAll(io.LimitReader(readmeFile, MaxReadmePreviewBytes))
			if err != nil {
				log.Errorf("Reading Readme: %s", err.Error())
				return nil, err
			}

			if len(ds.Readme.ScriptBytes) == MaxReadmePreviewBytes {
				ds.Readme.ScriptBytes = append(ds.Readme.ScriptBytes, []byte(`...`)...)
			}
			ds.Readme.SetScriptFile(nil)
		}
	}

	if err = ds.OpenBodyFile(ctx, fs); err != nil {
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
