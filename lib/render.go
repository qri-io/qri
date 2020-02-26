package lib

import (
	"context"
	"fmt"
	"net/rpc"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

// RenderRequests encapsulates business logic for this node's
// user profile
// TODO (b5): switch to using an Instance instead of separate fields
type RenderRequests struct {
	cli  *rpc.Client
	repo repo.Repo
}

// NewRenderRequests creates a RenderRequests pointer from either a repo
// or an rpc.Client
func NewRenderRequests(r repo.Repo, cli *rpc.Client) *RenderRequests {
	if r != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewRenderRequests"))
	}

	return &RenderRequests{
		cli:  cli,
		repo: r,
	}
}

// CoreRequestsName implements the Requets interface
func (RenderRequests) CoreRequestsName() string { return "render" }

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
func (r *RenderRequests) RenderViz(p *RenderParams, res *[]byte) (err error) {
	if r.cli != nil {
		return r.cli.Call("RenderRequests.RenderViz", p, res)
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

	if err = repo.CanonicalizeDatasetRef(r.repo, &ref); err == repo.ErrNotFound {
		return fmt.Errorf("unknown dataset '%s'", ref.AliasString())
	} else if err != nil {
		return err
	}

	*res, err = base.Render(ctx, r.repo, ref, p.Template)
	return err
}

// RenderReadme renders the readme into html for the given dataset
func (r *RenderRequests) RenderReadme(p *RenderParams, res *string) (err error) {
	if r.cli != nil {
		return r.cli.Call("RenderRequests.RenderReadme", p, res)
	}
	ctx := context.TODO()

	if err = p.Validate(); err != nil {
		return err
	}

	var ds *dataset.Dataset
	if p.Dataset != nil {
		ds = p.Dataset
	} else {
		ref, err := base.ToDatasetRef(p.Ref, r.repo, p.UseFSI)
		if err != nil {
			return err
		}

		if p.UseFSI {
			if ref.FSIPath == "" {
				return fsi.ErrNoLink
			}
			if ds, err = fsi.ReadDir(ref.FSIPath); err != nil {
				return fmt.Errorf("loading linked dataset: %s", err)
			}
		} else {
			ds, err = dsfs.LoadDataset(ctx, r.repo.Store(), ref.Path)
			if err != nil {
				return fmt.Errorf("loading dataset: %s", err)
			}
		}
	}

	if ds.Readme == nil {
		return fmt.Errorf("no readme to render")
	}

	if err = ds.Readme.OpenScriptFile(ctx, r.repo.Filesystem()); err != nil {
		return err
	}
	if ds.Readme.ScriptFile() == nil {
		return fmt.Errorf("no readme to render")
	}

	*res, err = base.RenderReadme(ctx, ds.Readme.ScriptFile())
	return err
}
