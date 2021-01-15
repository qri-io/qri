package lib

import (
	"context"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/dsref"
)

// TransformMethods encapsulates business logic for transforms
type TransformMethods struct {
	inst *Instance
}

// CoreRequestsName implements the Requests interface
func (TransformMethods) CoreRequestsName() string { return "apply" }

// NewTransformMethods creates a TransformMethods pointer from a qri instance
func NewTransformMethods(inst *Instance) *TransformMethods {
	return &TransformMethods{
		inst: inst,
	}
}

// ApplyParams are parameters for the apply command
type ApplyParams struct {
	Refstr    string
	Transform *dataset.Transform
	Source    string
	Secrets   map[string]string

	// TODO(dustmop): add `Wait bool`, if false, run the save asynchronously
	// and return events on the bus that provide the progress of the save operation

	ScriptOutput io.Writer
}

// Valid returns an error if ApplyParams fields are in an invalid state
func (p *ApplyParams) Valid() error {
	if p.Refstr == "" && p.Transform == nil {
		return fmt.Errorf("one or both of Reference, Transform are required")
	}
	return nil
}

// ApplyResult is the result of an apply command
type ApplyResult struct {
	Data *dataset.Dataset
	// TODO(dustmop): Make Apply asynchronous, running the transform in a go-routine.
	// Return a channel that will send progress on the execution.
}

// Apply runs a transform script
func (m *TransformMethods) Apply(p *ApplyParams, res *ApplyResult) error {
	err := p.Valid()
	if err != nil {
		return err
	}

	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("TransformMethods.Apply", p, res))
	}

	ctx := context.TODO()
	ref := dsref.Ref{}
	if p.Refstr != "" {
		ref, _, err = m.inst.ParseAndResolveRefWithWorkingDir(ctx, p.Refstr, "")
		if err != nil {
			return err
		}
	}

	ds := &dataset.Dataset{}
	if !ref.IsEmpty() {
		ds.Name = ref.Name
		ds.Peername = ref.Username
	}
	if p.Transform != nil {
		ds.Transform = p.Transform
		ds.Transform.OpenScriptFile(ctx, m.inst.repo.Filesystem())
	}

	r := m.inst.Repo()
	str := m.inst.node.LocalStreams
	loader := NewParseResolveLoadFunc("", m.inst.defaultResolver(), m.inst)

	scriptOut := p.ScriptOutput
	err = base.TransformApply(ctx, ds, r, loader, str, scriptOut, p.Secrets)
	if err != nil {
		return err
	}

	if err = base.InlineJSONBody(ds); err != nil && err != base.ErrNoBodyToInline {
		return err
	}
	res.Data = ds
	return nil
}
