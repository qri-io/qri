package fsi

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/fsi/component"
)

// GetBody is an FSI version of actions.GetBody
func GetBody(dirPath string, format dataset.DataFormat, fcfg dataset.FormatConfig, offset, limit int, all bool) ([]byte, error) {

	components, err := component.ListDirectoryComponents(dirPath)
	if err != nil {
		return nil, err
	}

	err = component.ExpandListedComponents(components, nil)
	if err != nil {
		return nil, err
	}

	bodyComponent := components.Base().GetSubcomponent("body")
	f, err := os.Open(bodyComponent.Base().SourceFile)
	if err != nil {
		return nil, err
	}

	stComponent := components.Base().GetSubcomponent("structure")
	stComponent.LoadAndFill(nil)
	structure, ok := stComponent.(*component.StructureComponent)
	if !ok {
		return nil, fmt.Errorf("could not get structure")
	}
	stValue := structure.Value

	defer f.Close()
	file := qfs.NewMemfileReader(filepath.Base(bodyComponent.Base().SourceFile), f)

	st := &dataset.Structure{}
	assign := &dataset.Structure{
		Format: format.String(),
		Schema: stValue.Schema,
	}
	if fcfg != nil {
		assign.FormatConfig = fcfg.Map()
	}
	st.Assign(stValue, assign)

	return base.ConvertBodyFile(file, stValue, st, limit, offset, all)
}
