package fsi

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
)

// GetBody is an FSI version of actions.GetBody
func GetBody(dirPath string, format dataset.DataFormat, fcfg dataset.FormatConfig, offset, limit int, all bool) ([]byte, error) {
	ds, mapping, _, err := ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	bodyPath, ok := mapping["body"]
	if !ok {
		return nil, fmt.Errorf("no body found")
	}

	f, err := os.Open(filepath.Join(dirPath, bodyPath))
	if err != nil {
		return nil, err
	}

	defer f.Close()
	file := qfs.NewMemfileReader(filepath.Base(bodyPath), f)

	st := &dataset.Structure{}
	assign := &dataset.Structure{
		Format: format.String(),
		Schema: ds.Structure.Schema,
	}
	if fcfg != nil {
		assign.FormatConfig = fcfg.Map()
	}
	st.Assign(ds.Structure, assign)

	return base.ConvertBodyFile(file, ds.Structure, st, limit, offset, all)
}
