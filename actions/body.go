package actions

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qri/p2p"
)

// LookupBody grabs a subset of a dataset's body
func LookupBody(node *p2p.QriNode, path string, format dataset.DataFormat, fcfg dataset.FormatConfig, limit, offset int, all bool) (bodyPath string, data []byte, err error) {
	var (
		file  cafs.File
		store = node.Repo.Store()
	)

	ds, err := dsfs.LoadDataset(store, datastore.NewKey(path))
	if err != nil {
		log.Debug(err.Error())
		return "", nil, err
	}

	file, err = dsfs.LoadBody(store, ds)
	if err != nil {
		log.Debug(err.Error())
		return "", nil, err
	}

	st := &dataset.Structure{}
	st.Assign(ds.Structure, &dataset.Structure{
		Format:       format,
		FormatConfig: fcfg,
		Schema:       ds.Structure.Schema,
	})

	data, err = ConvertBodyFile(file, ds.Structure, st, limit, offset, all)
	if err != nil {
		log.Debug(err.Error())
		return "", nil, err
	}

	return ds.BodyPath, data, nil
}

// ConvertBodyFile takes an input file & structure, and converts a specified selection
// to the structure specified by out
func ConvertBodyFile(file cafs.File, in, out *dataset.Structure, limit, offset int, all bool) (data []byte, err error) {
	buf, err := dsio.NewEntryBuffer(out)
	if err != nil {
		err = fmt.Errorf("error allocating result buffer: %s", err)
		return
	}
	rr, err := dsio.NewEntryReader(in, file)
	if err != nil {
		err = fmt.Errorf("error allocating data reader: %s", err)
		return
	}

	if !all {
		rr = &dsio.PagedReader{
			Reader: rr,
			Limit:  limit,
			Offset: offset,
		}
	}
	err = dsio.Copy(rr, buf)

	if err := buf.Close(); err != nil {
		return nil, fmt.Errorf("error closing row buffer: %s", err.Error())
	}

	return buf.Bytes(), nil
}
