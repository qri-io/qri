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

	var structure *dataset.Structure

	var schema map[string]interface{}
	stComponent := components.Base().GetSubcomponent("structure")
	if stComponent != nil {
		stComponent.LoadAndFill(nil)
		comp, ok := stComponent.(*component.StructureComponent)
		if !ok {
			return nil, fmt.Errorf("could not get structure")
		}
		structure = comp.Value
		schema = structure.Schema
	}

	if schema == nil {
		bodyFormat := bodyComponent.Base().Format
		// If there was no structure, define one using the body's file extension.
		if structure == nil {
			structure = &dataset.Structure{}
			structure.Format = bodyFormat
		}
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
	st.Assign(structure, assign)

	return base.ConvertBodyFile(file, structure, st, limit, offset, all)
}
