// Package ds exposes the qri dataset document model into starlark
package ds

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"sync"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qfs"
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

// MutateFieldCheck is a function to check if a dataset field can be mutated
// before mutating a field, dataset will call MutateFieldCheck with as specific
// a path as possible and bail if an error is returned
type MutateFieldCheck func(path ...string) error

// Dataset is a qri dataset starlark type
type Dataset struct {
	frozen    bool
	read      *dataset.Dataset
	write     *dataset.Dataset
	bodyCache starlark.Iterable
	check     MutateFieldCheck
	modBody   bool
}

// compile-time interface assertions
var (
	_ starlark.Value    = (*Dataset)(nil)
	_ starlark.HasAttrs = (*Dataset)(nil)
)

// methods defined on the dataset object
var dsMethods = map[string]*starlark.Builtin{
	"set_meta":      starlark.NewBuiltin("set_meta", dsSetMeta),
	"get_meta":      starlark.NewBuiltin("get_meta", dsGetMeta),
	"get_structure": starlark.NewBuiltin("get_structure", dsGetStructure),
	"set_structure": starlark.NewBuiltin("set_structure", dsSetStructure),
	"get_body":      starlark.NewBuiltin("get_body", dsGetBody),
	"set_body":      starlark.NewBuiltin("set_body", dsSetBody),
}

// NewDataset creates a dataset object, intended to be called from go-land to prepare datasets
// for handing to other functions
func NewDataset(ds *dataset.Dataset, check MutateFieldCheck) *Dataset {
	return &Dataset{read: ds, check: check}
}

// New creates a new dataset from starlark land
func New(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	d := &Dataset{read: &dataset.Dataset{}, write: &dataset.Dataset{}}
	return d, nil
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
	return builtinAttr(d, name, dsMethods)
}

// AttrNames lists available attributes
func (d *Dataset) AttrNames() []string {
	return builtinAttrNames(dsMethods)
}

// SetMutable assigns an underlying dataset that can be mutated
func (d *Dataset) SetMutable(ds *dataset.Dataset) {
	d.write = ds
}

// IsBodyModified returns whether the body has been modified by set_body
func (d *Dataset) IsBodyModified() bool {
	return d.modBody
}

// checkCanMutateField runs the check function if one is defined
func (d *Dataset) checkCanMutateField(path ...string) error {
	if d.check != nil {
		return d.check(path...)
	}
	return nil
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

	var provider *dataset.Meta
	if self.read != nil && self.read.Meta != nil {
		provider = self.read.Meta
	}
	if self.write != nil && self.write.Meta != nil {
		provider = self.write.Meta
	}

	if provider == nil {
		return starlark.None, nil
	}

	data, err := json.Marshal(provider)
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
	if self.write == nil {
		return starlark.None, fmt.Errorf("cannot call set_meta on read-only dataset")
	}

	key := keyx.GoString()

	if err := self.checkCanMutateField("meta", "key"); err != nil {
		return starlark.None, err
	}

	val, err := util.Unmarshal(valx)
	if err != nil {
		return nil, err
	}

	if self.write.Meta == nil {
		self.write.Meta = &dataset.Meta{}
	}

	return starlark.None, self.write.Meta.Set(key, val)
}

// dsGetStructure gets a dataset structure component
func dsGetStructure(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	self := b.Receiver().(*Dataset)

	var provider *dataset.Structure
	if self.read != nil && self.read.Structure != nil {
		provider = self.read.Structure
	}
	if self.write != nil && self.write.Structure != nil {
		provider = self.write.Structure
	}

	if provider == nil {
		return starlark.None, nil
	}

	data, err := json.Marshal(provider)
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
	if self.write == nil {
		return starlark.None, fmt.Errorf("cannot call set_structure on read-only dataset")
	}

	if err := self.checkCanMutateField("structure"); err != nil {
		return starlark.None, err
	}

	val, err := util.Unmarshal(valx)
	if err != nil {
		return starlark.None, err
	}

	if self.write.Structure == nil {
		self.write.Structure = &dataset.Structure{}
	}

	data, err := json.Marshal(val)
	if err != nil {
		return starlark.None, err
	}

	err = json.Unmarshal(data, self.write.Structure)
	return starlark.None, err
}

// dsGetBody returns the body of the dataset we're transforming. The read version is returned until
// the dataset is modified by set_body, then the write version is returned instead.
func dsGetBody(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	self := b.Receiver().(*Dataset)

	if self.bodyCache != nil {
		return self.bodyCache, nil
	}

	var valx starlark.Value
	if err := starlark.UnpackArgs("get_body", args, kwargs, "default?", &valx); err != nil {
		return starlark.None, err
	}

	var provider *dataset.Dataset
	if self.read != nil {
		provider = self.read
	}
	if self.modBody && self.write != nil {
		provider = self.write
	}

	if provider.BodyFile() == nil {
		if valx == nil {
			return starlark.None, nil
		}
		return valx, nil
	}

	if provider.Structure == nil {
		return starlark.None, fmt.Errorf("error: no structure for dataset")
	}

	// TODO - this is bad. make not bad.
	data, err := ioutil.ReadAll(provider.BodyFile())
	if err != nil {
		return starlark.None, err
	}
	provider.SetBodyFile(qfs.NewMemfileBytes("body.json", data))

	rr, err := dsio.NewEntryReader(provider.Structure, qfs.NewMemfileBytes("body.json", data))
	if err != nil {
		return starlark.None, fmt.Errorf("error allocating data reader: %s", err)
	}
	w, err := NewStarlarkEntryWriter(provider.Structure)
	if err != nil {
		return starlark.None, fmt.Errorf("error allocating starlark entry writer: %s", err)
	}

	err = dsio.Copy(rr, w)
	if err != nil {
		return starlark.None, err
	}
	if err = w.Close(); err != nil {
		return starlark.None, err
	}

	if iter, ok := w.Value().(starlark.Iterable); ok {
		self.bodyCache = iter
		return self.bodyCache, nil
	}
	return starlark.None, fmt.Errorf("value is not iterable")
}

// dsSetBody assigns the dataset body. Future calls to GetBody will return this newly mutated body,
// even if assigned value is the same as what was already there.
func dsSetBody(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		data    starlark.Value
		parseAs starlark.String
	)
	if err := starlark.UnpackArgs("set_body", args, kwargs, "data", &data, "parse_as?", &parseAs); err != nil {
		return starlark.None, err
	}
	self := b.Receiver().(*Dataset)

	if self.frozen {
		return starlark.None, fmt.Errorf("cannot call set_body on frozen dataset")
	}
	if self.write == nil {
		return starlark.None, fmt.Errorf("cannot call set_body on read-only dataset")
	}
	if err := self.checkCanMutateField("body"); err != nil {
		return starlark.None, err
	}
	if err := self.checkCanMutateField("structure"); err != nil {
		err = fmt.Errorf("cannot use a transform to set the body of a dataset and manually adjust structure at the same time")
		return starlark.None, err
	}

	if df := parseAs.GoString(); df != "" {
		if _, err := dataset.ParseDataFormatString(df); err != nil {
			return starlark.None, fmt.Errorf("invalid parse_as format: %q", df)
		}

		str, ok := data.(starlark.String)
		if !ok {
			return starlark.None, fmt.Errorf("expected data for %q format to be a string", df)
		}

		self.write.SetBodyFile(qfs.NewMemfileBytes(fmt.Sprintf("body.%s", df), []byte(string(str))))
		self.modBody = true
		self.bodyCache = nil

		if err := detect.Structure(self.write); err != nil {
			return nil, err
		}

		return starlark.None, nil
	}

	iter, ok := data.(starlark.Iterable)
	if !ok {
		return starlark.None, fmt.Errorf("expected body data to be iterable")
	}

	self.write.Structure = self.writeStructure(data)

	w, err := dsio.NewEntryBuffer(self.write.Structure)
	if err != nil {
		return starlark.None, err
	}
	r := NewEntryReader(self.write.Structure, iter)
	if err := dsio.Copy(r, w); err != nil {
		return starlark.None, err
	}
	if err := w.Close(); err != nil {
		return starlark.None, err
	}

	self.write.SetBodyFile(qfs.NewMemfileBytes(fmt.Sprintf("body.%s", self.write.Structure.Format), w.Bytes()))
	self.modBody = true
	self.bodyCache = nil

	return starlark.None, nil
}

// writeStructure determines the destination data structure for writing a
// dataset body, falling back to a default json structure based on input values
// if no prior structure exists
func (d *Dataset) writeStructure(data starlark.Value) *dataset.Structure {
	// if the write structure has been set, use that
	if d.write != nil && d.write.Structure != nil {
		return d.write.Structure
	}

	// fall back to inheriting from read structure
	if d.read != nil && d.read.Structure != nil {
		return d.read.Structure
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
