package actions

import (
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
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

	data, err = base.ConvertBodyFile(file, ds.Structure, st, limit, offset, all)
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	return data, nil
}
