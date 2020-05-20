package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
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
	OutFormat string
}

// Validate checks if render parameters are valid
func (p *RenderParams) Validate() error {
	if p.Ref != "" && p.Dataset != nil {
		return fmt.Errorf("cannot provide both a reference and a dataset to render")
	}
	return nil
}

// RenderViz renders a viz component as html
func (m *RenderMethods) RenderViz(p *RenderParams, res *[]byte) (err error) {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("RenderMethods.RenderViz", p, res))
	}
	ctx := context.TODO()

	if err = p.Validate(); err != nil {
		return err
	}

	if p.Dataset != nil {
		return fmt.Errorf("rendering dynamic dataset viz component is not supported")
	}

	var ref reporef.DatasetRef
	if ref, err = repo.ParseDatasetRef(p.Ref); err != nil {
		return
	}

	if err = repo.CanonicalizeDatasetRef(m.inst.repo, &ref); err == repo.ErrNotFound {
		return fmt.Errorf("unknown dataset '%s'", ref.AliasString())
	} else if err != nil {
		return err
	}

	*res, err = base.Render(ctx, m.inst.repo, ref, p.Template)
	return err
}

// RenderReadme renders the readme into html for the given dataset
func (m *RenderMethods) RenderReadme(p *RenderParams, res *string) (err error) {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("RenderMethods.RenderReadme", p, res))
	}
	ctx := context.TODO()

	if err = p.Validate(); err != nil {
		return err
	}

	var ds *dataset.Dataset
	if p.Dataset != nil {
		ds = p.Dataset
	} else {
		ref, _, err := m.inst.ParseAndResolveRefWithWorkingDir(ctx, p.Ref, "local")
		if err != nil {
			return err
		}

		ds, err = m.inst.loadDataset(ctx, ref)
		if err != nil {
			return fmt.Errorf("loading dataset: %w", err)
		}
	}

	if ds.Readme == nil {
		return fmt.Errorf("no readme to render")
	}

	if err = ds.Readme.OpenScriptFile(ctx, m.inst.repo.Filesystem()); err != nil {
		return err
	}
	if ds.Readme.ScriptFile() == nil {
		return fmt.Errorf("no readme to render")
	}

	*res, err = base.RenderReadme(ctx, ds.Readme.ScriptFile())
	return err
}
