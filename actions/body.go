package actions

import (
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/fs"
	"github.com/qri-io/qri/p2p"
)

// GetBody grabs some or all of a dataset's body, writing an output in the desired format
func GetBody(node *p2p.QriNode, ds *dataset.Dataset, format dataset.DataFormat, fcfg dataset.FormatConfig, limit, offset int, all bool) (data []byte, err error) {
	if ds == nil {
		return nil, fmt.Errorf("can't load body from a nil dataset")
	}

	file := ds.BodyFile()
	if file == nil {
		err = fmt.Errorf("no body file to read")
		return
	}

	st := &dataset.Structure{}
	assign := &dataset.Structure{
		Format: format.String(),
		Schema: ds.Structure.Schema,
	}
	if fcfg != nil {
		assign.FormatConfig = fcfg.Map()
	}
	st.Assign(ds.Structure, assign)

	data, err = ConvertBodyFile(file, ds.Structure, st, limit, offset, all)
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	return data, nil
}

// ConvertBodyFile takes an input file & structure, and converts a specified selection
// to the structure specified by out
func ConvertBodyFile(file fs.File, in, out *dataset.Structure, limit, offset int, all bool) (data []byte, err error) {
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
