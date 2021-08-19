// Package ds exposes the qri dataset document model into starlark
package ds

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"sort"
	"sync"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/tabular"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/starlib/dataframe"
	"github.com/qri-io/starlib/util"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// ModuleName defines the expected name for this Module when used
// in starlark's load() function, eg: load('dataset.star', 'dataset')
const ModuleName = "dataset.star"

var (
	once          sync.Once
	datasetModule starlark.StringDict
)

// LoadModule loads the base64 module.
// It is concurrency-safe and idempotent.
func LoadModule() (starlark.StringDict, error) {
	once.Do(func() {
		datasetModule = starlark.StringDict{
			"dataset": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"new": starlark.NewBuiltin("new", New),
			}),
		}
	})
	return datasetModule, nil
}

// Dataset is a qri dataset starlark type
type Dataset struct {
	frozen    bool
	ds        *dataset.Dataset
	bodyFrame starlark.Value
	changes   map[string]struct{}
}

// compile-time interface assertions
var (
	_ starlark.Value       = (*Dataset)(nil)
	_ starlark.HasAttrs    = (*Dataset)(nil)
	_ starlark.HasSetField = (*Dataset)(nil)
)

// methods defined on the dataset object
var dsMethods = map[string]*starlark.Builtin{
	"set_meta":      starlark.NewBuiltin("set_meta", dsSetMeta),
	"get_meta":      starlark.NewBuiltin("get_meta", dsGetMeta),
	"get_structure": starlark.NewBuiltin("get_structure", dsGetStructure),
	"set_structure": starlark.NewBuiltin("set_structure", dsSetStructure),
}

// NewDataset creates a dataset object, intended to be called from go-land to prepare datasets
// for handing to other functions
func NewDataset(ds *dataset.Dataset) *Dataset {
	return &Dataset{ds: ds, changes: make(map[string]struct{})}
}

// New creates a new dataset from starlark land
func New(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	d := &Dataset{ds: &dataset.Dataset{}, changes: make(map[string]struct{})}
	return d, nil
}

// Changes returns a map of which components have been changed
func (d *Dataset) Changes() map[string]struct{} {
	return d.changes
}

// String returns the Dataset as a string
func (d *Dataset) String() string {
	return d.stringify()
}

// Type returns a short string describing the value's type.
func (Dataset) Type() string { return fmt.Sprintf("%s.Dataset", "dataset") }

// Freeze renders Dataset immutable.
func (d *Dataset) Freeze() { d.frozen = true }

// Hash cannot be used with Dataset
func (d *Dataset) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable: %s", d.Type())
}

// Truth converts the dataset into a bool
func (d *Dataset) Truth() starlark.Bool {
	return true
}

// Attr gets a value for a string attribute
func (d *Dataset) Attr(name string) (starlark.Value, error) {
	if name == "body" {
		return d.getBody()
	}
	return builtinAttr(d, name, dsMethods)
}

// AttrNames lists available attributes
func (d *Dataset) AttrNames() []string {
	return append(builtinAttrNames(dsMethods), "body")
}

// SetField assigns to a field of the DataFrame
func (d *Dataset) SetField(name string, val starlark.Value) error {
	if d.frozen {
		return fmt.Errorf("cannot set, Dataset is frozen")
	}
	if name == "body" {
		return d.setBody(val)
	}
	return starlark.NoSuchAttrError(name)
}

func (d *Dataset) stringify() string {
	// TODO(dustmop): Improve the stringification of a Dataset
	return "<Dataset>"
}

func builtinAttr(recv starlark.Value, name string, methods map[string]*starlark.Builtin) (starlark.Value, error) {
	b := methods[name]
	if b == nil {
		return nil, nil // no such method
	}
	return b.BindReceiver(recv), nil
}

func builtinAttrNames(methods map[string]*starlark.Builtin) []string {
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// dsGetMeta gets a dataset meta component
func dsGetMeta(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	self := b.Receiver().(*Dataset)

	if self.ds.Meta == nil {
		return starlark.None, nil
	}

	data, err := json.Marshal(self.ds.Meta)
	if err != nil {
		return starlark.None, err
	}

	jsonData := map[string]interface{}{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return starlark.None, err
	}

	return util.Marshal(jsonData)
}

// dsSetMeta sets a dataset meta field
func dsSetMeta(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		keyx starlark.String
		valx starlark.Value
	)
	if err := starlark.UnpackPositionalArgs("set_meta", args, kwargs, 2, &keyx, &valx); err != nil {
		return nil, err
	}
	self := b.Receiver().(*Dataset)

	if self.frozen {
		return starlark.None, fmt.Errorf("cannot call set_meta on frozen dataset")
	}
	self.changes["meta"] = struct{}{}

	key := keyx.GoString()

	val, err := util.Unmarshal(valx)
	if err != nil {
		return nil, err
	}

	if self.ds.Meta == nil {
		self.ds.Meta = &dataset.Meta{}
	}

	return starlark.None, self.ds.Meta.Set(key, val)
}

// dsGetStructure gets a dataset structure component
func dsGetStructure(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	self := b.Receiver().(*Dataset)

	if self.ds.Structure == nil {
		return starlark.None, nil
	}

	data, err := json.Marshal(self.ds.Structure)
	if err != nil {
		return starlark.None, err
	}

	jsonData := map[string]interface{}{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return starlark.None, err
	}

	return util.Marshal(jsonData)
}

// SetStructure sets the dataset structure component
func dsSetStructure(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	self := b.Receiver().(*Dataset)

	var valx starlark.Value
	if err := starlark.UnpackPositionalArgs("set_structure", args, kwargs, 1, &valx); err != nil {
		return nil, err
	}

	if self.frozen {
		return starlark.None, fmt.Errorf("cannot call set_structure on frozen dataset")
	}
	self.changes["structure"] = struct{}{}

	val, err := util.Unmarshal(valx)
	if err != nil {
		return starlark.None, err
	}

	if self.ds.Structure == nil {
		self.ds.Structure = &dataset.Structure{}
	}

	data, err := json.Marshal(val)
	if err != nil {
		return starlark.None, err
	}

	err = json.Unmarshal(data, self.ds.Structure)
	return starlark.None, err
}

func (d *Dataset) getBody() (starlark.Value, error) {
	if d.bodyFrame != nil {
		return d.bodyFrame, nil
	}

	bodyfile := d.ds.BodyFile()
	if bodyfile == nil {
		// If no body exists, return an empty data frame
		df, _ := dataframe.NewDataFrame(nil, nil, nil)
		d.bodyFrame = df
		return df, nil
	}

	if d.ds.Structure == nil {
		return starlark.None, fmt.Errorf("error: no structure for dataset")
	}

	// TODO(dustmop): DataFrame should be able to work with an
	// efficient, streaming body file.
	data, err := ioutil.ReadAll(d.ds.BodyFile())
	if err != nil {
		return starlark.None, err
	}
	d.ds.SetBodyFile(qfs.NewMemfileBytes("body.json", data))

	rr, err := dsio.NewEntryReader(d.ds.Structure, qfs.NewMemfileBytes("body.json", data))
	if err != nil {
		return starlark.None, fmt.Errorf("error allocating data reader: %s", err)
	}

	entries, err := base.ReadEntries(rr)
	if err != nil {
		return starlark.None, err
	}
	rows := [][]interface{}{}
	eachEntry := entries.([]interface{})
	for _, ent := range eachEntry {
		r := ent.([]interface{})
		rows = append(rows, r)
	}

	df, err := dataframe.NewDataFrame(rows, nil, nil)
	if err != nil {
		return nil, err
	}
	d.bodyFrame = df
	return df, nil
}

func (d *Dataset) setBody(val starlark.Value) error {
	df, err := dataframe.NewDataFrame(val, nil, nil)
	if err != nil {
		return err
	}
	d.bodyFrame = df
	return nil
}

// writeStructure determines the destination data structure for writing a
// dataset body, falling back to a default json structure based on input values
// if no prior structure exists
func (d *Dataset) writeStructure(data starlark.Value) *dataset.Structure {
	// if the write structure has been set, use that
	if d.ds != nil && d.ds.Structure != nil {
		return d.ds.Structure
	}

	// use a default of json as a last resort
	sch := dataset.BaseSchemaArray
	if data.Type() == "dict" {
		sch = dataset.BaseSchemaObject
	}

	return &dataset.Structure{
		Format: "json",
		Schema: sch,
	}
}

// AssignBodyFromDataframe converts the DataFrame on the object into
// a proper dataset.bodyfile
func (d *Dataset) AssignBodyFromDataframe() error {
	if d.ds == nil {
		return nil
	}
	if d.bodyFrame == nil {
		return nil
	}
	df, ok := d.bodyFrame.(*dataframe.DataFrame)
	if !ok {
		return fmt.Errorf("bodyFrame has invalid type %v", reflect.TypeOf(d.bodyFrame))
	}

	st := d.ds.Structure
	if st == nil {
		st = &dataset.Structure{
			Format: "json",
			Schema: tabular.BaseTabularSchema,
		}
	}

	w, err := dsio.NewEntryBuffer(st)
	if err != nil {
		return err
	}

	for i := 0; i < df.NumRows(); i++ {
		w.WriteEntry(dsio.Entry{Index: i, Value: df.Row(i)})
	}
	if err := w.Close(); err != nil {
		return err
	}
	d.ds.SetBodyFile(qfs.NewMemfileBytes(fmt.Sprintf("body.%s", st.Format), w.Bytes()))
	err = detect.Structure(d.ds)
	if err != nil {
		return err
	}
	return nil
}
