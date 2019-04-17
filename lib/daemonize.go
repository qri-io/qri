package lib

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/p2p"
)

// DaemonizeRequests encapsulates business logic of daemonize operation
type DaemonizeRequests struct {
	cli  *rpc.Client
	node *p2p.QriNode
}

// CoreRequestsName implements the Requests interface
func (DaemonizeRequests) CoreRequestsName() string { return "daemonize" }

// NewDaemonizeRequests creates a DaemonizeRequests pointer from either a node or an rpc.Client
func NewDaemonizeRequests(node *p2p.QriNode, cli *rpc.Client) *DaemonizeRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewDaemonizeRequests"))
	}

	return &DaemonizeRequests{
		node: node,
		cli:  cli,
	}
}

// Daemonize handles actions about daemonization
func (r *DaemonizeRequests) Daemonize(params *DaemonizeParams, out *bool) error {
	switch params.Action {
	case "":
		err := actions.DaemonHelp()
		if err != nil {
			return err
		}
	case "install":
		if r.cli != nil {
			return fmt.Errorf("cannot daemonize if `qri connect` is already running")
		}

		err := actions.DaemonInstall()
		if err != nil {
			return err
		}
	case "show":
		err := actions.DaemonShow()
		if err != nil {
			return err
		}
	case "uninstall":
		err := actions.DaemonUninstall()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown daemonize action: \"%s\"", params.Action)
	}

	*out = true
	return nil
}

// DaemonizeParams holds parameters for daemonize commands
type DaemonizeParams struct {
	Action string
}
