package lib

import (
	"context"
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/base"
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
	Ref            string
	Template       []byte
	TemplateFormat string
}

// Render executes a template against a template
func (r *RenderRequests) Render(p *RenderParams, res *[]byte) (err error) {
	if r.cli != nil {
		return r.cli.Call("RenderRequests.Render", p, res)
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
