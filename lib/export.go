package lib

import (
	"context"
	"fmt"
	"net/rpc"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/archive"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// ExportRequests encapsulates business logic of export operation
// TODO (b5): switch to using an Instance instead of separate fields
type ExportRequests struct {
	node *p2p.QriNode
	cli  *rpc.Client
}

// CoreRequestsName implements the Requests interface
func (r ExportRequests) CoreRequestsName() string { return "export" }

// NewExportRequests creates a ExportRequests pointer from either a repo
// or an rpc.Client
func NewExportRequests(node *p2p.QriNode, cli *rpc.Client) *ExportRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both node and client supplied to NewExportRequests"))
	}
	return &ExportRequests{
		node: node,
		cli:  cli,
	}
}

// ExportParams defines parameters for the export method
type ExportParams struct {
	Ref       string
	TargetDir string
	Output    string
	Format    string
	Zipped    bool
}

// Export exports a dataset in the specified format
func (r *ExportRequests) Export(p *ExportParams, fileWritten *string) (err error) {
	if p.TargetDir == "" {
		p.TargetDir = "."
		if err = qfs.AbsPath(&p.TargetDir); err != nil {
			return err
		}
	}

	if r.cli != nil {
		return r.cli.Call("ExportRequests.Export", p, fileWritten)
	}
	ctx := context.TODO()

	if p.Ref == "" {
		return repo.ErrEmptyRef
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return fmt.Errorf("'%s' is not a valid dataset reference", p.Ref)
	}
	if err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		return err
	}

	ds, err := base.ReadDatasetPath(ctx, r.node.Repo, ref.String())
	if err != nil {
		return fmt.Errorf("reading dataset '%s': %w", ref, err)
	}

	*fileWritten, err = archive.Export(ctx, r.node.Repo.Store(), ds, ref.String(), p.TargetDir, p.Output, p.Format, p.Zipped)
	return err
}
