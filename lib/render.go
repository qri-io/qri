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
	Ref       string
	Template  []byte
	UseFSI    bool
	OutFormat string
}

// RenderTemplate executes a template against a dataset
func (r *RenderRequests) RenderTemplate(p *RenderParams, res *[]byte) (err error) {
	if r.cli != nil {
		return r.cli.Call("RenderRequests.RenderTemplate", p, res)
	}
	ctx := context.TODO()

	var ref repo.DatasetRef
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

	ref, err := base.ToDatasetRef(p.Ref, r.repo, p.UseFSI)
	if err != nil {
		return err
	}

	var ds *dataset.Dataset
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

	err = ds.Readme.OpenScriptFile(ctx, r.repo.Filesystem())
	if err != nil {
		return err
	}

	*res, err = base.RenderReadme(ctx, ds.Readme.ScriptFile())
	return err
}
