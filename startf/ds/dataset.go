// Package ds exposes the qri dataset document model into starlark
package ds

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/qri-io/dataset"
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
	read      *dataset.Dataset
	write     *dataset.Dataset
	bodyCache starlark.Iterable
	check     MutateFieldCheck
	modBody   bool
}

// NewDataset creates a dataset object, intended to be called from go-land to prepare datasets
// for handing to other functions
func NewDataset(ds *dataset.Dataset, check MutateFieldCheck) *Dataset {
	return &Dataset{read: ds, check: check}
}

// New creates a new dataset from starlark land
func New(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	d := &Dataset{read: &dataset.Dataset{}, write: &dataset.Dataset{}}
	return d.Methods(), nil
}

// SetMutable assigns an underlying dataset that can be mutated
func (d *Dataset) SetMutable(ds *dataset.Dataset) {
	d.write = ds
}

// IsBodyModified returns whether the body has been modified by set_body
func (d *Dataset) IsBodyModified() bool {
	return d.modBody
}

// Methods exposes dataset methods as starlark values
func (d *Dataset) Methods() *starlarkstruct.Struct {
	return starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
		"set_meta":      starlark.NewBuiltin("set_meta", d.SetMeta),
		"get_meta":      starlark.NewBuiltin("get_meta", d.GetMeta),
		"get_structure": starlark.NewBuiltin("get_structure", d.GetStructure),
		"set_structure": starlark.NewBuiltin("set_structure", d.SetStructure),
		"get_body":      starlark.NewBuiltin("get_body", d.GetBody),
		"set_body":      starlark.NewBuiltin("set_body", d.SetBody),
	})
}

// checkField runs the check function if one is defined
func (d *Dataset) checkField(path ...string) error {
	if d.check != nil {
		return d.check(path...)
	}
	return nil
}

// GetMeta gets a dataset meta component
func (d *Dataset) GetMeta(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var provider *dataset.Meta
	if d.read != nil && d.read.Meta != nil {
		provider = d.read.Meta
	}
	if d.write != nil && d.write.Meta != nil {
		provider = d.write.Meta
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

// SetMeta sets a dataset meta field
func (d *Dataset) SetMeta(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		keyx starlark.String
		valx starlark.Value
	)
	if err := starlark.UnpackPositionalArgs("set_meta", args, kwargs, 2, &keyx, &valx); err != nil {
		return nil, err
	}

	if d.write == nil {
		return starlark.None, fmt.Errorf("cannot call set_meta on read-only dataset")
	}

	key := keyx.GoString()

	if err := d.checkField("meta", "key"); err != nil {
		return starlark.None, err
	}

	val, err := util.Unmarshal(valx)
	if err != nil {
		return nil, err
	}

	if d.write.Meta == nil {
		d.write.Meta = &dataset.Meta{}
	}

	return starlark.None, d.write.Meta.Set(key, val)
}

// GetStructure gets a dataset structure component
func (d *Dataset) GetStructure(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var provider *dataset.Structure
	if d.read != nil && d.read.Structure != nil {
		provider = d.read.Structure
	}
	if d.write != nil && d.write.Structure != nil {
		provider = d.write.Structure
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
func (d *Dataset) SetStructure(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var valx starlark.Value
	if err := starlark.UnpackPositionalArgs("set_structure", args, kwargs, 1, &valx); err != nil {
		return nil, err
	}

	if d.write == nil {
		return starlark.None, fmt.Errorf("cannot call set_structure on read-only dataset")
	}

	if err := d.checkField("structure"); err != nil {
		return starlark.None, err
	}

	val, err := util.Unmarshal(valx)
	if err != nil {
		return starlark.None, err
	}

	if d.write.Structure == nil {
		d.write.Structure = &dataset.Structure{}
	}

	data, err := json.Marshal(val)
	if err != nil {
		return starlark.None, err
	}

	err = json.Unmarshal(data, d.write.Structure)
	return starlark.None, err
}

// GetBody returns the body of the dataset we're transforming. The read version is returned until
// the dataset is modified by set_body, then the write version is returned instead.
func (d *Dataset) GetBody(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if d.bodyCache != nil {
		return d.bodyCache, nil
	}

	var valx starlark.Value
	if err := starlark.UnpackArgs("get_body", args, kwargs, "default?", &valx); err != nil {
		return starlark.None, err
	}

	var provider *dataset.Dataset
	if d.read != nil {
		provider = d.read
	}
	if d.modBody && d.write != nil {
		provider = d.write
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
		d.bodyCache = iter
		return d.bodyCache, nil
	}
	return starlark.None, fmt.Errorf("value is not iterable")
}

// SetBody assigns the dataset body. Future calls to GetBody will return this newly mutated body,
// even if assigned value is the same as what was already there.
func (d *Dataset) SetBody(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		data    starlark.Value
		parseAs starlark.String
	)

	if err := starlark.UnpackArgs("set_body", args, kwargs, "data", &data, "parse_as?", &parseAs); err != nil {
		return starlark.None, err
	}

	if d.write == nil {
		return starlark.None, fmt.Errorf("cannot call set_body on read-only dataset")
	}

	if err := d.checkField("body"); err != nil {
		return starlark.None, err
	}

	if err := d.checkField("structure"); err != nil {
		err = fmt.Errorf("cannot use a transform to set the body of a dataset and manually adjust structure at the same time")
		return starlark.None, err
	}

	df := parseAs.GoString()
	if df != "" {
		if _, err := dataset.ParseDataFormatString(df); err != nil {
			return starlark.None, fmt.Errorf("invalid parse_as format: '%s'", df)
		}

		str, ok := data.(starlark.String)
		if !ok {
			return starlark.None, fmt.Errorf("expected data for '%s' format to be a string", df)
		}

		d.write.SetBodyFile(qfs.NewMemfileBytes(fmt.Sprintf("body.%s", df), []byte(string(str))))
		d.modBody = true
		d.bodyCache = nil
		return starlark.None, nil
	}

	iter, ok := data.(starlark.Iterable)
	if !ok {
		return starlark.None, fmt.Errorf("expected body data to be iterable")
	}

	d.write.Structure = d.writeStructure(data)

	w, err := dsio.NewEntryBuffer(d.write.Structure)
	if err != nil {
		return starlark.None, err
	}

	r := NewEntryReader(d.write.Structure, iter)
	if err := dsio.Copy(r, w); err != nil {
		return starlark.None, err
	}
	if err := w.Close(); err != nil {
		return starlark.None, err
	}

	d.write.SetBodyFile(qfs.NewMemfileBytes(fmt.Sprintf("body.%s", d.write.Structure.Format), w.Bytes()))
	d.modBody = true
	d.bodyCache = nil

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
