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
	defer f.Close()

	stComponent := components.Base().GetSubcomponent("structure")
	stComponent.LoadAndFill(nil)
	structure, ok := stComponent.(*component.StructureComponent)
	if !ok {
		return nil, fmt.Errorf("could not get structure")
	}
	stValue := structure.Value
	schema := stValue.Schema

	if schema == nil {
		bodyFormat := bodyComponent.Base().Format
		// Create schema by detecting it from the body.
		// TODO(dlong): This should move into `dsio` package.
		entries, err := component.OpenEntryReader(f, bodyFormat)
		if err != nil {
			return nil, err
		}
		schema = entries.Structure().Schema
		// Reset the reader
		f.Seek(0, 0)
	}

	file := qfs.NewMemfileReader(filepath.Base(bodyComponent.Base().SourceFile), f)

	st := &dataset.Structure{}
	assign := &dataset.Structure{
		Format: format.String(),
		Schema: schema,
	}
	if fcfg != nil {
		assign.FormatConfig = fcfg.Map()
	}
	st.Assign(stValue, assign)

	return base.ConvertBodyFile(file, stValue, st, limit, offset, all)
}
