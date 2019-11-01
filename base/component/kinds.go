package component

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/fill"
	"gopkg.in/yaml.v2"
)

// FilesysComponent represents a collection of components existing as files on a filesystem
type FilesysComponent struct {
	BaseComponent
}

// Compare compares to another component
func (fc *FilesysComponent) Compare(compare Component) (bool, error) {
	return false, fmt.Errorf("cannot compare filesys component containers")
}

// IsEmpty returns whether the component collection is empty
func (fc *FilesysComponent) IsEmpty() bool {
	return len(fc.Subcomponents) == 0
}

// WriteTo writes the component as a file to the directory
func (fc *FilesysComponent) WriteTo(dirPath string) error {
	return fmt.Errorf("cannot write filesys component")
}

// RemoveFrom removes the component file from the directory
func (fc *FilesysComponent) RemoveFrom(dirPath string) error {
	return fmt.Errorf("cannot write filesys component")
}

// DropDerivedValues drops derived values from the component
func (fc *FilesysComponent) DropDerivedValues() {
	for compName := range fc.BaseComponent.Subcomponents {
		fc.BaseComponent.Subcomponents[compName].DropDerivedValues()
	}
}

// LoadAndFill loads data from the component source file and assigngs it
func (fc *FilesysComponent) LoadAndFill(ds *dataset.Dataset) error {
	return nil
}

// StructuredData cannot be returned for a filesystem
func (fc *FilesysComponent) StructuredData() (interface{}, error) {
	return nil, fmt.Errorf("cannot convert filesys to a structured data")
}

// DatasetComponent represents a dataset with components
type DatasetComponent struct {
	BaseComponent
	Value *dataset.Dataset
}

// Compare compares to another component
func (dc *DatasetComponent) Compare(compare Component) (bool, error) {
	other, ok := compare.(*DatasetComponent)
	if !ok {
		return false, nil
	}
	if err := dc.LoadAndFill(nil); err != nil {
		return false, err
	}
	if err := compare.LoadAndFill(nil); err != nil {
		return false, err
	}
	return compareComponentData(dc.Value, other.Value)
}

// WriteTo writes the component as a file to the directory
func (dc *DatasetComponent) WriteTo(dirPath string) error {
	return fmt.Errorf("cannot write dataset component")
}

// RemoveFrom removes the component file from the directory
func (dc *DatasetComponent) RemoveFrom(dirPath string) error {
	return fmt.Errorf("cannot write dataset component")
}

// DropDerivedValues drops derived values from the component
func (dc *DatasetComponent) DropDerivedValues() {
	for compName := range dc.BaseComponent.Subcomponents {
		if compName == "dataset" {
			continue
		}
		dc.BaseComponent.Subcomponents[compName].DropDerivedValues()
	}
	if dc.Value != nil {
		dc.Value.DropDerivedValues()
	}
}

// LoadAndFill loads data from the component source file and assigngs it
func (dc *DatasetComponent) LoadAndFill(ds *dataset.Dataset) error {
	return nil
}

// StructuredData returns the dataset as a map[string]
func (dc *DatasetComponent) StructuredData() (interface{}, error) {
	if err := dc.LoadAndFill(nil); err != nil {
		return nil, err
	}
	return structToMap(dc.Value)
}

func structToMap(value interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// MetaComponent represents a meta component
type MetaComponent struct {
	BaseComponent
	Value *dataset.Meta
}

// Compare compares to another component
func (mc *MetaComponent) Compare(compare Component) (bool, error) {
	other, ok := compare.(*MetaComponent)
	if !ok {
		return false, nil
	}
	if err := mc.LoadAndFill(nil); err != nil {
		return false, err
	}
	if err := compare.LoadAndFill(nil); err != nil {
		return false, err
	}
	return compareComponentData(mc.Value, other.Value)
}

// WriteTo writes the component as a file to the directory
func (mc *MetaComponent) WriteTo(dirPath string) error {
	if err := mc.LoadAndFill(nil); err != nil {
		return err
	}
	// Okay to output an empty meta, we do so for `qri init`.
	if mc.Value != nil {
		return writeComponentFile(mc.Value, dirPath, "meta.json")
	}
	return nil
}

// RemoveFrom removes the component file from the directory
func (mc *MetaComponent) RemoveFrom(dirPath string) error {
	// TODO(dlong): Does component have SourceFile set?
	if err := os.Remove(filepath.Join(dirPath, "meta.json")); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// DropDerivedValues drops derived values from the component
func (mc *MetaComponent) DropDerivedValues() {
	mc.Value.DropDerivedValues()
}

// LoadAndFill loads data from the component source file and assigngs it
func (mc *MetaComponent) LoadAndFill(ds *dataset.Dataset) error {
	if mc.Base().IsLoaded {
		return nil
	}
	if mc.Value != nil {
		mc.Base().IsLoaded = true
		return nil
	}
	fields, err := mc.Base().LoadFile()
	if err != nil {
		return err
	}
	mc.Value = &dataset.Meta{}
	if err := fill.Struct(fields, mc.Value); err != nil {
		return err
	}
	if ds != nil {
		ds.Meta = mc.Value
	}
	return nil
}

// StructuredData returns the meta as a map[string]
func (mc *MetaComponent) StructuredData() (interface{}, error) {
	if err := mc.LoadAndFill(nil); err != nil {
		return nil, err
	}
	return structToMap(mc.Value)
}

// StructureComponent represents a structure component
type StructureComponent struct {
	BaseComponent
	Value           *dataset.Structure
	SchemaInference func(*dataset.Dataset) (map[string]interface{}, error)
}

// Compare compares to another component
func (sc *StructureComponent) Compare(compare Component) (bool, error) {
	other, ok := compare.(*StructureComponent)
	if !ok {
		return false, nil
	}
	if err := sc.LoadAndFill(nil); err != nil {
		return false, err
	}
	if err := compare.LoadAndFill(nil); err != nil {
		return false, err
	}
	// TODO(dlong): DropDerivedValues should not be used here, but lazy evaluation requires it.
	sc.Value.DropDerivedValues()
	other.Value.DropDerivedValues()
	return compareComponentData(sc.Value, other.Value)
}

// WriteTo writes the component as a file to the directory
func (sc *StructureComponent) WriteTo(dirPath string) error {
	if err := sc.LoadAndFill(nil); err != nil {
		return err
	}
	if sc.Value != nil && !sc.Value.IsEmpty() {
		return writeComponentFile(sc.Value, dirPath, "structure.json")
	}
	return nil
}

// RemoveFrom removes the component file from the directory
func (sc *StructureComponent) RemoveFrom(dirPath string) error {
	if err := os.Remove(filepath.Join(dirPath, "structure.json")); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// DropDerivedValues drops derived values from the component
func (sc *StructureComponent) DropDerivedValues() {
	sc.Value.DropDerivedValues()
}

// LoadAndFill loads data from the component source file and assigngs it
func (sc *StructureComponent) LoadAndFill(ds *dataset.Dataset) error {
	if sc.Base().IsLoaded {
		return nil
	}
	if sc.Value != nil {
		sc.Base().IsLoaded = true
		return nil
	}
	fields, err := sc.Base().LoadFile()
	if err != nil {
		return err
	}
	sc.Value = &dataset.Structure{}
	if err := fill.Struct(fields, sc.Value); err != nil {
		return err
	}
	if sc.Value.Schema == nil && sc.SchemaInference != nil {
		// do nothing, don't infer schema to insert here
	}
	if ds != nil {
		ds.Structure = sc.Value
	}
	return nil
}

// StructuredData returns the structure as a map[string]
func (sc *StructureComponent) StructuredData() (interface{}, error) {
	if err := sc.LoadAndFill(nil); err != nil {
		return nil, err
	}
	return structToMap(sc.Value)
}

// CommitComponent represents a commit component
type CommitComponent struct {
	BaseComponent
	Value *dataset.Commit
}

// Compare compares to another component
func (cc *CommitComponent) Compare(compare Component) (bool, error) {
	other, ok := compare.(*CommitComponent)
	if !ok {
		return false, nil
	}
	if err := cc.LoadAndFill(nil); err != nil {
		return false, err
	}
	if err := compare.LoadAndFill(nil); err != nil {
		return false, err
	}
	return compareComponentData(cc.Value, other.Value)
}

// WriteTo writes the component as a file to the directory
func (cc *CommitComponent) WriteTo(dirPath string) error {
	if err := cc.LoadAndFill(nil); err != nil {
		return err
	}
	if cc.Value != nil && !cc.Value.IsEmpty() {
		return writeComponentFile(cc.Value, dirPath, "commit.json")
	}
	return nil
}

// RemoveFrom removes the component file from the directory
func (cc *CommitComponent) RemoveFrom(dirPath string) error {
	if err := os.Remove(filepath.Join(dirPath, "commit.json")); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// DropDerivedValues drops derived values from the component
func (cc *CommitComponent) DropDerivedValues() {
	cc.Value.DropDerivedValues()
}

// LoadAndFill loads data from the component source file and assigngs it
func (cc *CommitComponent) LoadAndFill(ds *dataset.Dataset) error {
	if cc.Base().IsLoaded {
		return nil
	}
	if cc.Value != nil {
		cc.Base().IsLoaded = true
		return nil
	}
	fields, err := cc.Base().LoadFile()
	if err != nil {
		return err
	}
	cc.Value = &dataset.Commit{}
	if err := fill.Struct(fields, cc.Value); err != nil {
		return err
	}
	if ds != nil {
		ds.Commit = cc.Value
	}
	return nil
}

// StructuredData returns the commit as a map[string]
func (cc *CommitComponent) StructuredData() (interface{}, error) {
	if err := cc.LoadAndFill(nil); err != nil {
		return nil, err
	}
	return structToMap(cc.Value)
}

// BodyComponent represents a body component
type BodyComponent struct {
	BaseComponent
	Resolver       qfs.PathResolver
	BodyFile       qfs.File
	Structure      *dataset.Structure
	InferredSchema map[string]interface{}
	Value          interface{}
}

// NewBodyComponent returns a body component for the given source file
func NewBodyComponent(file string) *BodyComponent {
	return &BodyComponent{
		BaseComponent: BaseComponent{
			SourceFile: file,
			Format:     filepath.Ext(file),
		},
	}
}

// Compare compares to another component
func (bc *BodyComponent) Compare(compare Component) (bool, error) {
	other, ok := compare.(*BodyComponent)
	if !ok {
		return false, nil
	}
	if err := bc.LoadAndFill(nil); err != nil {
		return false, err
	}
	if err := other.LoadAndFill(nil); err != nil {
		return false, err
	}
	return compareComponentData(bc.Value, other.Value)
}

// DropDerivedValues drops derived values from the component
func (bc *BodyComponent) DropDerivedValues() {
}

// LoadAndFill loads data from the component source file and assigngs it
func (bc *BodyComponent) LoadAndFill(ds *dataset.Dataset) error {
	if bc.Value != nil {
		return nil
	}

	var err error
	var entries dsio.EntryReader

	// TODO(dlong): Move this condition into a utility function in dataset.dsio.
	// TODO(dlong): Should we pipe ctx into this function, instead of using context.Background?
	if bc.BodyFile != nil {
		bf := bc.BodyFile
		entries, err = dsio.NewEntryReader(bc.Structure, bf)
		if err != nil {
			return err
		}
	} else if bc.Resolver != nil {
		bf, err := bc.Resolver.Get(context.Background(), bc.Base().SourceFile)
		if err != nil {
			return err
		}
		entries, err = dsio.NewEntryReader(bc.Structure, bf)
		if err != nil {
			return err
		}
	} else {
		f, err := os.Open(bc.SourceFile)
		if err != nil {
			return err
		}
		entries, err = OpenEntryReader(f, bc.BaseComponent.Format)
		if err != nil {
			return err
		}
		bc.InferredSchema = entries.Structure().Schema
	}

	topLevel, err := dsio.GetTopLevelType(entries.Structure())
	if err != nil {
		return err
	}

	if topLevel == "array" {
		result := make([]interface{}, 0)
		for {
			ent, err := entries.ReadEntry()
			if err != nil {
				if err.Error() == io.EOF.Error() {
					break
				}
				return err
			}
			result = append(result, ent.Value)
		}
		bc.Value = result
	} else {
		result := make(map[string]interface{})
		for {
			ent, err := entries.ReadEntry()
			if err != nil {
				if err.Error() == io.EOF.Error() {
					break
				}
				return err
			}
			result[ent.Key] = ent.Value
		}
		bc.Value = result
	}

	return nil
}

// StructuredData returns the body as a map[string] or []interface{}, depending on top-level type
func (bc *BodyComponent) StructuredData() (interface{}, error) {
	if err := bc.LoadAndFill(nil); err != nil {
		return nil, err
	}
	return bc.Value, nil
}

// WriteTo writes the component as a file to the directory
func (bc *BodyComponent) WriteTo(dirPath string) error {
	if bc.Value == nil {
		err := bc.LoadAndFill(nil)
		if err != nil {
			return err
		}
	}
	body := bc.Value
	if bc.Structure == nil {
		return fmt.Errorf("cannot write body without a structure")
	}
	data, err := SerializeBody(body, bc.Structure)
	if err != nil {
		return err
	}
	bodyFilename := fmt.Sprintf("body.%s", bc.Format)
	return ioutil.WriteFile(filepath.Join(dirPath, bodyFilename), data, os.ModePerm)
}

// RemoveFrom removes the component file from the directory
func (bc *BodyComponent) RemoveFrom(dirPath string) error {
	bodyFilename := fmt.Sprintf("body.%s", bc.Format)
	if err := os.Remove(filepath.Join(dirPath, bodyFilename)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// OpenEntryReader opens a entry reader for the file, determining the schema automatically
// TODO(dlong): Move this to dataset.dsio
func OpenEntryReader(file *os.File, format string) (dsio.EntryReader, error) {
	st := dataset.Structure{Format: format}
	schema, _, err := detect.Schema(&st, file)
	if err != nil {
		return nil, err
	}
	file.Seek(0, 0)
	st.Schema = schema
	entries, err := dsio.NewEntryReader(&st, file)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// SerializeBody writes the source, which must be an array or object,
// TODO(dlong): Move this to dataset.dsio
func SerializeBody(source interface{}, st *dataset.Structure) ([]byte, error) {
	buff := bytes.Buffer{}
	writer, err := dsio.NewEntryWriter(st, &buff)
	if err != nil {
		return nil, err
	}
	switch data := source.(type) {
	case []interface{}:
		for i, val := range data {
			writer.WriteEntry(dsio.Entry{Index: i, Value: val})
		}
	case map[string]interface{}:
		for key, val := range data {
			writer.WriteEntry(dsio.Entry{Key: key, Value: val})
		}
	}
	writer.Close()
	return buff.Bytes(), nil
}

// ReadmeComponent represents a meta component
type ReadmeComponent struct {
	BaseComponent
	Resolver qfs.PathResolver
	Value    *dataset.Readme
}

// Compare compares to another component
func (rc *ReadmeComponent) Compare(compare Component) (bool, error) {
	other, ok := compare.(*ReadmeComponent)
	if !ok {
		return false, nil
	}
	if err := rc.LoadAndFill(nil); err != nil {
		return false, err
	}
	if err := compare.LoadAndFill(nil); err != nil {
		return false, err
	}
	return compareComponentData(rc.Value, other.Value)
}

// WriteTo writes the component as a file to the directory
func (rc *ReadmeComponent) WriteTo(dirPath string) error {
	if err := rc.LoadAndFill(nil); err != nil {
		return err
	}
	if rc.Value != nil && !rc.Value.IsEmpty() {
		filename := filepath.Join(dirPath, fmt.Sprintf("readme.%s", rc.Format))
		if err := ioutil.WriteFile(filename, rc.Value.ScriptBytes, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

// RemoveFrom removes the component file from the directory
func (rc *ReadmeComponent) RemoveFrom(dirPath string) error {
	// TODO(dlong): Does component have SourceFile set?
	if err := os.Remove(filepath.Join(dirPath, fmt.Sprintf("readme.%s", rc.Format))); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// DropDerivedValues drops derived values from the component
func (rc *ReadmeComponent) DropDerivedValues() {
	rc.Value.DropDerivedValues()
}

// LoadAndFill loads data from the component source file and assigns it
func (rc *ReadmeComponent) LoadAndFill(ds *dataset.Dataset) error {
	if rc.Base().IsLoaded {
		return nil
	}
	if rc.Value == nil {
		fields, err := rc.Base().LoadFile()
		if err != nil {
			return err
		}
		rc.Value = &dataset.Readme{}
		if err := fill.Struct(fields, rc.Value); err != nil {
			return err
		}
	}
	rc.Base().IsLoaded = true

	if rc.Resolver != nil {
		err := rc.Value.InlineScriptFile(context.Background(), rc.Resolver)
		if err != nil {
			return err
		}
	}

	if ds != nil {
		ds.Readme = rc.Value
	}
	return nil
}

// StructuredData returns the readme as a map[string]
func (rc *ReadmeComponent) StructuredData() (interface{}, error) {
	if err := rc.LoadAndFill(nil); err != nil {
		return nil, err
	}
	return structToMap(rc.Value)
}

// Base returns the common base data for the component
func (bc *BaseComponent) Base() *BaseComponent {
	return bc
}

// LoadFile opens the source file for the component and unmarshals it, adds errors for duplicate
// components and parse errors
func (bc *BaseComponent) LoadFile() (map[string]interface{}, error) {
	if bc.IsLoaded {
		return nil, nil
	}

	data, err := ioutil.ReadFile(bc.SourceFile)
	if err != nil {
		bc.SetErrorAsProblem("file-open", err)
		return nil, err
	}
	bc.IsLoaded = true
	// Parse the file bytes using the specified format
	fields := make(map[string]interface{})
	switch bc.Format {
	case "json":
		if err = json.Unmarshal(data, &fields); err != nil {
			bc.SetErrorAsProblem("parse", err)
			return nil, err
		}
		return fields, nil
	case "yaml":
		if err = yaml.Unmarshal(data, &fields); err != nil {
			bc.SetErrorAsProblem("parse", err)
			return nil, err
		}
		return fields, nil
	case "html", "md":
		fields["ScriptBytes"] = data
		return fields, nil
	}
	return nil, fmt.Errorf("unknown local file format \"%s\"", bc.Format)
}

// SetErrorAsProblem converts the error into a problem and assigns it
func (bc *BaseComponent) SetErrorAsProblem(kind string, err error) {
	bc.ProblemKind = kind
	bc.ProblemMessage = err.Error()
}

// GetSubcomponent returns the component with the given name
func (bc *BaseComponent) GetSubcomponent(name string) Component {
	return bc.Subcomponents[name]
}

// SetSubcomponent constructs a component of the appropriate type and adds it as a subcomponent
func (bc *BaseComponent) SetSubcomponent(name string, base BaseComponent) Component {
	var component Component
	if name == "meta" {
		component = &MetaComponent{BaseComponent: base}
	} else if name == "commit" {
		component = &CommitComponent{BaseComponent: base}
	} else if name == "structure" {
		component = &StructureComponent{BaseComponent: base}
	} else if name == "readme" {
		component = &ReadmeComponent{BaseComponent: base}
	} else if name == "body" {
		component = &BodyComponent{BaseComponent: base}
	} else if name == "dataset" {
		component = &DatasetComponent{BaseComponent: base}
	} else {
		return nil
	}
	if bc.Subcomponents == nil {
		bc.Subcomponents = make(map[string]Component)
	}
	bc.Subcomponents[name] = component
	return component
}

// RemoveSubcomponent removes the component with the given name
func (bc *BaseComponent) RemoveSubcomponent(name string) {
	delete(bc.Subcomponents, name)
}

func compareComponentData(first interface{}, second interface{}) (bool, error) {
	left, err := json.Marshal(first)
	if err != nil {
		return false, err
	}
	rite, err := json.Marshal(second)
	if err != nil {
		return false, err
	}
	return string(left) == string(rite), nil
}

func writeComponentFile(value interface{}, dirPath string, basefile string) error {
	data, err := json.MarshalIndent(value, "", " ")
	if err != nil {
		return err
	}
	// TODO(dlong): How does this relate to Base.SourceFile? Should respect that.
	err = ioutil.WriteFile(filepath.Join(dirPath, basefile), data, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
