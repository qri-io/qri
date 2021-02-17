package lib

import (
	"context"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/transform"
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
	Secrets   map[string]string
	Wait      bool

	Source string
	// TODO(arqu): substitute with websockets when working over the wire
	ScriptOutput io.Writer `json:"-"`
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
	Data  *dataset.Dataset
	RunID string `json:"runID"`
}

// Apply runs a transform script
func (m *TransformMethods) Apply(ctx context.Context, p *ApplyParams) (*ApplyResult, error) {
	err := p.Valid()
	if err != nil {
		return nil, err
	}

	res := &ApplyResult{}

	if m.inst.http != nil {
		err = m.inst.http.Call(ctx, AEApply, p, res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	ref := dsref.Ref{}
	if p.Refstr != "" {
		ref, _, err = m.inst.ParseAndResolveRefWithWorkingDir(ctx, p.Refstr, "")
		if err != nil {
			return nil, err
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

	str := m.inst.node.LocalStreams
	loader := NewParseResolveLoadFunc("", m.inst.defaultResolver(), m.inst)

	// allocate an ID for the transform, for now just log the events it produces
	runID := transform.NewRunID()
	m.inst.bus.SubscribeID(func(ctx context.Context, e event.Event) error {
		go func() {
			log.Debugw("apply transform event", "type", e.Type, "payload", e.Payload)
			if e.Type == event.ETTransformPrint {
				if msg, ok := e.Payload.(event.TransformMessage); ok {
					if p.ScriptOutput != nil {
						io.WriteString(p.ScriptOutput, msg.Msg)
						io.WriteString(p.ScriptOutput, "\n")
					}
				}
			}
		}()
		return nil
	}, runID)

	scriptOut := p.ScriptOutput
	err = m.inst.transform.Apply(ctx, ds, loader, runID, m.inst.bus, p.Wait, str, scriptOut, p.Secrets)
	if err != nil {
		return nil, err
	}

	if p.Wait {
		if err = base.InlineJSONBody(ds); err != nil && err != base.ErrNoBodyToInline {
			return nil, err
		}
		res.Data = ds
	}
	res.RunID = runID
	return res, nil
}
