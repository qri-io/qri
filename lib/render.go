package lib

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/repo"
)

// RenderMethods execute qri dataset viz components to a visual representation
type RenderMethods interface {
	Methods
	Render(p *RenderParams, res *[]byte) error
}

// NewRenderMethods creates a RenderMethods from an instance
func NewRenderMethods(inst Instance) RenderMethods {
	return renderMethods{
		cli:  inst.RPC(),
		repo: inst.Repo(),
	}
}

// renderMethods encapsulates business logic for this node's
// user profile
type renderMethods struct {
	cli  *rpc.Client
	repo repo.Repo
}

// MethodsKind implements the Requets interface
func (renderMethods) MethodsKind() string { return "RenderMethods" }

// RenderParams defines parameters for the Render method
type RenderParams struct {
	Ref            string
	Template       []byte
	TemplateFormat string
}

// Render executes a template against a template
func (r renderMethods) Render(p *RenderParams, res *[]byte) (err error) {
	if r.cli != nil {
		return r.cli.Call("RenderMethods.Render", p, res)
	}

	var ref repo.DatasetRef
	if ref, err = repo.ParseDatasetRef(p.Ref); err != nil {
		return
	}

	if err = repo.CanonicalizeDatasetRef(r.repo, &ref); err == repo.ErrNotFound {
		return fmt.Errorf("unknown dataset '%s'", ref.AliasString())
	} else if err != nil {
		return err
	}

	if err := DefaultSelectedRef(r.repo, &ref); err != nil {
		return err
	}

	*res, err = base.Render(r.repo, ref, p.Template)
	return err
}
