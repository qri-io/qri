package component

import (
	"fmt"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
)

// Component represents one of two things, either a single component (meta, body), or a collection
// of components, such as an entire dataset or a directory of files that encode components.
type Component interface {
	Base() *BaseComponent
	Compare(Component) (bool, error)
	WriteTo(dirPath string) (targetFile string, err error)
	RemoveFrom(dirPath string) error
	DropDerivedValues()
	LoadAndFill(*dataset.Dataset) error
	StructuredData() (interface{}, error)
}

// NumberPossibleComponents is the number of subcomponents plus "dataset".
const NumberPossibleComponents = 8

// AllSubcomponentNames is the names of subcomponents that can live on a collection.
func AllSubcomponentNames() []string {
	return []string{"commit", "meta", "structure", "readme", "viz", "transform", "body"}
}

// ConvertDatasetToComponents will convert a dataset to a component collection
func ConvertDatasetToComponents(ds *dataset.Dataset, qfilesys qfs.Filesystem) Component {
	dc := DatasetComponent{}
	dc.Value = ds
	dc.Subcomponents = make(map[string]Component)

	dc.Subcomponents["dataset"] = &dc
	if ds.Meta != nil {
		mc := MetaComponent{}
		mc.Value = ds.Meta
		dc.Subcomponents["meta"] = &mc
	}
	if ds.Structure != nil {
		sc := StructureComponent{}
		sc.Value = ds.Structure
		dc.Subcomponents["structure"] = &sc
	}
	if ds.Commit != nil {
		cc := CommitComponent{}
		cc.Value = ds.Commit
		dc.Subcomponents["commit"] = &cc
	}
	if ds.Readme != nil {
		rc := ReadmeComponent{Resolver: qfilesys}
		rc.Value = ds.Readme
		rc.Format = "md"
		dc.Subcomponents["readme"] = &rc
	}
	if ds.Transform != nil {
		dc.Subcomponents["transform"] = &TransformComponent{
			BaseComponent: BaseComponent{Format: "star"},
			Resolver:      qfilesys,
			Value:         ds.Transform,
		}
	}

	if ds.Body != nil {
		bc := BodyComponent{Resolver: qfilesys}
		bc.Value = ds.Body
		bc.BodyFile = ds.BodyFile()
		bc.Structure = ds.Structure
		bc.Format = ds.Structure.Format
		dc.Subcomponents["body"] = &bc
	} else if ds.BodyPath != "" {
		bc := BodyComponent{Resolver: qfilesys}
		bc.SourceFile = ds.BodyPath
		bc.BodyFile = ds.BodyFile()
		bc.Structure = ds.Structure
		bc.Format = ds.Structure.Format
		dc.Subcomponents["body"] = &bc
	}

	return &dc
}

// ToDataset converts a component to a dataset. Should only be used on a
// component representing an entire dataset.
func ToDataset(comp Component) (*dataset.Dataset, error) {
	dsComp := comp.Base().GetSubcomponent("dataset")
	if dsComp == nil {
		comp := DatasetComponent{}
		comp.Value = &dataset.Dataset{}
		dsComp = &comp
	}
	dsCont, ok := dsComp.(*DatasetComponent)
	if !ok {
		return nil, fmt.Errorf("could not cast component to a Dataset")
	}
	ds := dsCont.Value
	if mdComponent := comp.Base().GetSubcomponent("meta"); mdComponent != nil {
		if err := mdComponent.LoadAndFill(ds); err != nil {
			return nil, err
		}
	}
	if cmComponent := comp.Base().GetSubcomponent("commit"); cmComponent != nil {
		if err := cmComponent.LoadAndFill(ds); err != nil {
			return nil, err
		}
	}
	if stComponent := comp.Base().GetSubcomponent("structure"); stComponent != nil {
		if err := stComponent.LoadAndFill(ds); err != nil {
			return nil, err
		}
	}
	if rmComponent := comp.Base().GetSubcomponent("readme"); rmComponent != nil {
		if err := rmComponent.LoadAndFill(ds); err != nil {
			return nil, err
		}
	}
	if tfComponent := comp.Base().GetSubcomponent("transform"); tfComponent != nil {
		if err := tfComponent.LoadAndFill(ds); err != nil {
			return nil, err
		}
	}
	if bdComponent := comp.Base().GetSubcomponent("body"); bdComponent != nil {
		if !bdComponent.Base().IsLoaded {
			ds.BodyPath = bdComponent.Base().SourceFile
		}
	}
	return ds, nil
}

// BaseComponent is the data elements common to any component
type BaseComponent struct {
	Subcomponents  map[string]Component
	ProblemKind    string
	ProblemMessage string
	// File information:
	ModTime    time.Time
	SourceFile string
	IsLoaded   bool
	Format     string
}
