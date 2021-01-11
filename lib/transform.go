package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/startf"
)

// TransformMethods encapsulates business logic for the qri search command
// TODO (b5): switch to using an Instance instead of separate fields
type TransformMethods struct {
	inst *Instance
}

// NewTransformMethods creates TransformMethods from a qri Instance
func NewTransformMethods(inst *Instance) *TransformMethods {
	return &TransformMethods{inst: inst}
}

// CoreRequestsName implements the requests
func (m TransformMethods) CoreRequestsName() string { return "transform" }

// ApplyParams encapsulates parameters to the Apply method
type ApplyParams struct {
	RefString string    `json:"refString"`
	Ref       dsref.Ref `json:"ref"`

	Transform *dataset.Transform `json:"transform"`
	Source    string             `json:"source"`
}

// Valid returns an error if ApplyParams fields are in an invalid state
func (p *ApplyParams) Valid() error {
	if p.RefString == "" && p.Ref.IsEmpty() && p.Transform == nil {
		return fmt.Errorf("one or both of Reference, Transform are required")
	}
	if p.RefString != "" && !p.Ref.IsEmpty() {
		return fmt.Errorf("cannot provide both a resolved reference and reference string")
	}

	return nil
}

// Apply executes a transform
func (m TransformMethods) Apply(p *ApplyParams, res *dataset.Dataset) error {
	err := p.Valid()
	if err != nil {
		return err
	}

	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("TransformMethods.Apply", p, res))
	}

	ctx := context.TODO()
	ref := p.Ref
	if ref.IsEmpty() && p.RefString != "" {
		ref, _, err = m.inst.ParseAndResolveRefWithWorkingDir(ctx, p.RefString, p.Source)
		if err != nil {
			return err
		}
	}

	// TODO (b5) - load previous dataset
	prev := &dataset.Dataset{}
	if !ref.IsEmpty() {
		prev, err = m.inst.LoadDataset(ctx, ref, p.Source)
		if err != nil {
			return err
		}
	}

	prlf, err := m.inst.NewParseResolveLoadFunc(p.Source)
	if err != nil {
		return err
	}

	if err = p.Transform.OpenScriptFile(ctx, m.inst.qfs); err != nil {
		return err
	}

	next := &dataset.Dataset{
		Transform: p.Transform,
	}
	err = startf.ExecScript(ctx, next, prev,
		startf.AddDatasetLoader(prlf),
		startf.SetSecrets(next.Transform.Secrets),
	)
	if err != nil {
		return err
	}

	*res = *next
	return nil
}
