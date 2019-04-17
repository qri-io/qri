package lib

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// LogRequests encapsulates business logic for the log
// of changes to datasets, think "git log"
// TODO (b5): switch to using an Instance instead of separate fields
type LogRequests struct {
	node *p2p.QriNode
	cli  *rpc.Client
}

// CoreRequestsName implements the Requets interface
func (r LogRequests) CoreRequestsName() string { return "log" }

// NewLogRequests creates a LogRequests pointer from either a repo
// or an rpc.Client
func NewLogRequests(node *p2p.QriNode, cli *rpc.Client) *LogRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both node and client supplied to NewLogRequests"))
	}
	return &LogRequests{
		node: node,
		cli:  cli,
	}
}

// LogParams defines parameters for the Log method
type LogParams struct {
	ListParams
	// Reference to data to fetch history for
	Ref repo.DatasetRef
}

// Log returns the history of changes for a given dataset
func (r *LogRequests) Log(params *LogParams, res *[]repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("LogRequests.Log", params, res)
	}

	ref := params.Ref
	if err = DefaultSelectedRef(r.node.Repo, &ref); err != nil {
		return
	}

	*res, err = actions.DatasetLog(r.node, ref, params.Limit, params.Offset)
	return
}
