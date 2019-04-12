package lib

import (
	"net/rpc"

	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// LogMethods shows a history of dataset versions
type LogMethods interface {
	Methods
	Log(params *LogParams, res *[]repo.DatasetRef) error
}

// NewLogMethods creates a logMethods pointer from either a repo
// or an rpc.Client
func NewLogMethods(inst Instance) LogMethods {
	return &logMethods{
		node: inst.Node(),
		cli:  inst.RPC(),
	}
}

// logMethods encapsulates business logic for the log
// of changes to datasets, think "git log"
type logMethods struct {
	node *p2p.QriNode
	cli  *rpc.Client
}

// MethodsKind implements the Requets interface
func (r logMethods) MethodsKind() string { return "LogMethods" }

// LogParams defines parameters for the Log method
type LogParams struct {
	ListParams
	// Reference to data to fetch history for
	Ref repo.DatasetRef
}

// Log returns the history of changes for a given dataset
func (r *logMethods) Log(params *LogParams, res *[]repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("LogMethods.Log", params, res)
	}

	ref := params.Ref
	if err = DefaultSelectedRef(r.node.Repo, &ref); err != nil {
		return
	}

	*res, err = actions.DatasetLog(r.node, ref, params.Limit, params.Offset)
	return
}
