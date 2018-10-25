package lib

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/repo"
)

// RenderRequests encapsulates business logic for this node's
// user profile
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
	Ref            repo.DatasetRef
	Template       []byte
	TemplateFormat string
	All            bool
	Limit, Offset  int
}

// Render executes a template against a template
func (r *RenderRequests) Render(p *RenderParams, res *[]byte) (err error) {
	if r.cli != nil {
		return r.cli.Call("RenderRequests.Render", p, res)
	}

	if err := DefaultSelectedRef(r.repo, &p.Ref); err != nil {
		return err
	}

	*res, err = base.Render(r.repo, p.Ref, p.Template, p.Limit, p.Offset, p.All)
	if err == repo.ErrNotFound {
		return NewError(err, fmt.Sprintf("could not find dataset '%s/%s'", p.Ref.Peername, p.Ref.Name))
	}

	return err
}
