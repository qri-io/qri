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
	"github.com/qri-io/qri/transform/run"
)

// TransformMethods groups together methods for transforms
type TransformMethods struct {
	d dispatcher
}

// Name returns the name of this method gropu
func (m *TransformMethods) Name() string {
	return "transform"
}

// Transform returns the TransformMethods that Instance has registered
func (inst *Instance) Transform() *TransformMethods {
	return &TransformMethods{d: inst}
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

// Validate returns an error if ApplyParams fields are in an invalid state
func (p *ApplyParams) Validate() error {
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
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "apply"), p)
	if res, ok := got.(*ApplyResult); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Implementations for transform methods follow

// transformImpl holds the method implementations for transforms
type transformImpl struct{}

// Apply runs a transform script
func (transformImpl) Apply(scp scope, p *ApplyParams) (*ApplyResult, error) {
	ctx := scp.Context()

	var err error
	ref := dsref.Ref{}
	if p.Refstr != "" {
		ref, _, err = scp.ParseAndResolveRefWithWorkingDir(ctx, p.Refstr, "")
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
		ds.Transform.OpenScriptFile(ctx, scp.Filesystem())
	}

	// allocate an ID for the transform, for now just log the events it produces
	runID := run.NewID()
	scp.Bus().SubscribeID(func(ctx context.Context, e event.Event) error {
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
	loader := scp.ParseResolveFunc()

	transformer := transform.NewTransformer(scp.AppContext(), loader, scp.Bus())
	err = transformer.Apply(ctx, ds, runID, p.Wait, scriptOut, p.Secrets)
	if err != nil {
		return nil, err
	}

	res := &ApplyResult{}
	if p.Wait {
		if err = base.InlineJSONBody(ds); err != nil && err != base.ErrNoBodyToInline {
			return nil, err
		}
		res.Data = ds
	}
	res.RunID = runID
	return res, nil
}
