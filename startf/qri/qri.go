package qri

import (
	"fmt"
	"sync"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/p2p"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// ModuleName defines the expected name for this module when used
// in starlark's load() function, eg: load('qri.star', 'qri')
const ModuleName = "qri.star"

var (
	once      sync.Once
	qriModule starlark.StringDict
)

// NewModule creates a new qri module instance
func NewModule(node *p2p.QriNode) *Module {
	return &Module{node: node}
}

// Module encapsulates state for a qri starlark module
type Module struct {
	node *p2p.QriNode
	ds   *dataset.Dataset
}

// Namespace produces this module's exported namespace
func (m *Module) Namespace() starlark.StringDict {
	return starlark.StringDict{
		"qri": m.Struct(),
	}
}

// Struct returns this module's methods as a starlark Struct
func (m *Module) Struct() *starlarkstruct.Struct {
	return starlarkstruct.FromStringDict(starlarkstruct.Default, m.AddAllMethods(starlark.StringDict{}))
}

// AddAllMethods augments a starlark.StringDict with all qri builtins. Should really only be used during "transform" step
func (m *Module) AddAllMethods(sd starlark.StringDict) starlark.StringDict {
	sd["list_datasets"] = starlark.NewBuiltin("list_datasets", m.ListDatasets)
	return sd
}

// ListDatasets shows current local datasets
func (m *Module) ListDatasets(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if m.node == nil {
		return starlark.None, fmt.Errorf("no qri node available to list datasets")
	}

	refs, err := m.node.Repo.References(1000, 0)
	if err != nil {
		return starlark.None, fmt.Errorf("error getting dataset list: %s", err.Error())
	}

	l := &starlark.List{}
	for _, ref := range refs {
		l.Append(starlark.String(ref.String()))
	}
	return l, nil
}
