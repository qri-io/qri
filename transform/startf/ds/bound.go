package ds

import (
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/starlib/dataframe"
	"go.starlark.net/starlark"
)

// BoundDataset represents the datset a transform script is bound to
type BoundDataset struct {
	frozen       bool
	commitCalled bool
	latest       *dataset.Dataset
	outconf      *dataframe.OutputConfig
	onCommit     func(ds *Dataset) error
	load         func(refstr string) (*Dataset, error)
}

// compile-time interface assertions
var (
	_ starlark.Value    = (*BoundDataset)(nil)
	_ starlark.HasAttrs = (*BoundDataset)(nil)
)

// NewBoundDataset constructs a target dataset
func NewBoundDataset(latest *dataset.Dataset, outconf *dataframe.OutputConfig, onCommit func(ds *Dataset) error) *BoundDataset {
	return &BoundDataset{latest: latest, onCommit: onCommit, outconf: outconf}
}

// String returns the Dataset as a string
func (b *BoundDataset) String() string { return b.stringify() }

// Type returns a short string describing the value's type.
func (BoundDataset) Type() string { return fmt.Sprintf("%s.BoundDataset", "dataset") }

// Freeze renders Dataset immutable.
func (b *BoundDataset) Freeze() { b.frozen = true }

// Hash cannot be used with Dataset
func (b *BoundDataset) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable: %s", b.Type())
}

// Truth converts the dataset into a bool
func (b *BoundDataset) Truth() starlark.Bool { return true }

// Attr gets a value for a string attribute
func (b *BoundDataset) Attr(name string) (starlark.Value, error) {
	return builtinAttr(b, name, boundDatasetMethods)
}

// AttrNames lists available attributes
func (b *BoundDataset) AttrNames() []string {
	return builtinAttrNames(boundDatasetMethods)
}

func (b *BoundDataset) stringify() string { return "<BoundDataset>" }

// methods defined on the history object
var boundDatasetMethods = map[string]*starlark.Builtin{
	"latest": starlark.NewBuiltin("latest", head),
	"commit": starlark.NewBuiltin("commit", commit),
}

func head(thread *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	self := builtin.Receiver().(*BoundDataset)
	return NewDataset(self.latest, self.outconf), nil
}

func commit(thread *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	self := builtin.Receiver().(*BoundDataset)
	if self.commitCalled {
		return nil, fmt.Errorf("commit can only be called once in a transform script")
	}
	starDs := &Dataset{}
	if err := starlark.UnpackArgs("commit", args, kwargs, "ds", starDs); err != nil {
		return starlark.None, err
	}
	if self.onCommit != nil {
		if err := self.onCommit(starDs); err != nil {
			return starlark.None, err
		}
	}

	self.commitCalled = true
	return starlark.None, nil
}
