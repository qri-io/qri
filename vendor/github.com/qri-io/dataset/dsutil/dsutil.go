// Dataset util funcs, placed here to avoid dataset package bloat
package dsutil

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// TODO - make sure a provided path is valid
func ValidPath(path datastore.Key) (datastore.Key, error) {
	return path, nil
}

// TODO - clean & find valid path to dataset
func ValidDatasetPath(path datastore.Key) (datastore.Key, error) {
	return path, nil
}

func WriteZipArchive(store cafs.Filestore, ds *dataset.Dataset, w io.Writer) error {
	zw := zip.NewWriter(w)

	dsf, err := zw.Create(dsfs.PackageFileDataset.String())
	if err != nil {
		return err
	}
	dsdata, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return err
	}
	_, err = dsf.Write(dsdata)
	if err != nil {
		return err
	}

	datadst, err := zw.Create(fmt.Sprintf("data.%s", ds.Structure.Format.String()))
	if err != nil {
		return err
	}

	datasrc, err := dsfs.LoadData(store, ds)
	if err != nil {
		return err
	}

	if _, err = io.Copy(datadst, datasrc); err != nil {
		return err
	}

	return zw.Close()
}

func WriteDir(store cafs.Filestore, ds *dataset.Dataset, path string) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	dsdata, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(path, dsfs.PackageFileDataset.String()), dsdata, os.ModePerm)
	if err != nil {
		return err
	}

	datasrc, err := dsfs.LoadData(store, ds)
	if err != nil {
		return err
	}

	datadst, err := os.Create(filepath.Join(path, fmt.Sprintf("data.%s", ds.Structure.Format.String())))
	if err != nil {
		return err
	}
	if _, err = io.Copy(datadst, datasrc); err != nil {
		return err
	}

	return nil
}
