package lib

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/dsref"
)

// RenderMethods encapsulates business logic for executing templates, using
// a dataset as a source
type RenderMethods struct {
	inst *Instance
}

// NewRenderMethods creates a RenderMethods pointer from either a repo
// or an rpc.Client
func NewRenderMethods(inst *Instance) *RenderMethods {
	return &RenderMethods{
		inst: inst,
	}
}

// CoreRequestsName implements the Requets interface
func (RenderMethods) CoreRequestsName() string { return "render" }

// RenderParams defines parameters for the Render method
type RenderParams struct {
	// Ref is a string reference to the dataset to render
	Ref string
	// Optionally pass an entire dataset in for rendering, if providing a dataset,
	// the Ref field must be empty
	Dataset *dataset.Dataset
	// Optional template override
	Template []byte
	// If true,
	UseFSI bool
	// Output format. defaults to "html"
	Format string
	// remote resolver to use
	Remote string
	// Old style viz component rendering
	Viz bool
}

// SetNonZeroDefaults assigns default values
func (p *RenderParams) SetNonZeroDefaults() {
	if p.Format == "" {
		p.Format = "html"
	}
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *RenderParams) UnmarshalFromRequest(r *http.Request) error {
	if p == nil {
		p = &RenderParams{}
	}

	params := *p
	if params.Ref == "" {
		params.Ref = r.FormValue("refstr")
	}

	_, err := dsref.Parse(params.Ref)
	if err != nil && params.Dataset == nil {
		return err
	}

	if !params.Viz {
		params.Viz = r.FormValue("viz") == "true"
	}
	if !params.UseFSI {
		params.UseFSI = r.FormValue("fsi") == "true"
	}

	if params.Remote == "" {
		params.Remote = r.FormValue("remote")
	}
	if params.Format == "" {
		params.Format = r.FormValue("format")
	}

	*p = params
	return nil
}

// Validate checks if render parameters are valid
func (p *RenderParams) Validate() error {
	if p.Ref != "" && p.Dataset != nil {
		return fmt.Errorf("cannot provide both a reference and a dataset to render")
	}
	return nil
}

// RenderViz renders a viz component as html
func (m *RenderMethods) RenderViz(ctx context.Context, p *RenderParams) ([]byte, error) {
	if m.inst.http != nil {
		var bres bytes.Buffer
		err := m.inst.http.CallRaw(ctx, AERender, p, &bres)
		if err != nil {
			return nil, err
		}
		return bres.Bytes(), nil
	}

	// TODO(dustmop): Remove this once scope is used here, Dispatch will call Validate
	if err := p.Validate(); err != nil {
		return nil, err
	}

	ds := p.Dataset
	if ds == nil {
		parseResolveLoad, err := m.inst.NewParseResolveLoadFunc(p.Remote)
		if err != nil {
			return nil, err
		}

		ds, err = parseResolveLoad(ctx, p.Ref)
		if err != nil {
			return nil, err
		}
	}

	res, err := base.Render(ctx, m.inst.repo, ds, p.Template)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// RenderReadme renders the readme into html for the given dataset
func (m *RenderMethods) RenderReadme(ctx context.Context, p *RenderParams) ([]byte, error) {
	if m.inst.http != nil {
		var bres bytes.Buffer
		err := m.inst.http.CallRaw(ctx, AERender, p, &bres)
		if err != nil {
			return nil, err
		}
		return bres.Bytes(), nil
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	var ds *dataset.Dataset
	if p.Dataset != nil {
		ds = p.Dataset
	} else {
		ref, source, err := m.inst.ParseAndResolveRefWithWorkingDir(ctx, p.Ref, "local")
		if err != nil {
			return nil, err
		}

		ds, err = m.inst.LoadDataset(ctx, ref, source)
		if err != nil {
			return nil, fmt.Errorf("loading dataset: %w", err)
		}
	}

	if ds.Readme == nil {
		return nil, fmt.Errorf("no readme to render")
	}

	if err := ds.Readme.OpenScriptFile(ctx, m.inst.repo.Filesystem()); err != nil {
		return nil, err
	}
	if ds.Readme.ScriptFile() == nil {
		return nil, fmt.Errorf("no readme to render")
	}

	res, err := base.RenderReadme(ctx, ds.Readme.ScriptFile())
	if err != nil {
		return nil, err
	}
	return []byte(res), nil
}
