package dsfs

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
)

// DerefMeta derferences a dataset's transform element if required
// should be a no-op if ds.Structure is nil or isn't a reference
func DerefMeta(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Meta != nil && ds.Meta.IsEmpty() && ds.Meta.Path != "" {
		md, err := loadMeta(ctx, store, ds.Meta.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("loading dataset metadata: %w", err)
		}
		md.Path = ds.Meta.Path
		ds.Meta = md
	}
	return nil
}

// loadMeta assumes the provided path is valid
func loadMeta(ctx context.Context, fs qfs.Filesystem, path string) (md *dataset.Meta, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("loading metadata file: %w", err)
	}
	md = &dataset.Meta{}
	err = md.UnmarshalJSON(data)
	return md, err
}

func addMetaFile(ds *dataset.Dataset, wfs *writeFiles) error {
	if ds.Meta == nil {
		return nil
	}

	ds.Meta.DropTransientValues()
	md, err := JSONFile(PackageFileMeta.Filename(), ds.Meta)
	if err != nil {
		return err
	}

	wfs.meta = md
	return nil
}
