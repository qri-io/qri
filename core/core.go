package core

import (
	"github.com/qri-io/qri/p2p"
)

// CoreRequests defines a set of core methods
type CoreRequests interface {
	// CoreRequestsName confirms participation in the CoreRequests interface while
	// also giving a human readable string for logging purposes
	CoreRequestsName() string
}

// Requests returns a slice of CoreRequests that defines the full local
// API of core methods
func Receivers(node *p2p.QriNode) []CoreRequests {
	r := node.Repo
	return []CoreRequests{
		NewDatasetRequests(r, nil),
		NewHistoryRequests(r),
		NewPeerRequests(r, node),
		NewProfileRequests(r),
		NewQueryRequests(r),
		NewSearchRequests(r),
	}
}

// func RemoteClient(addr string) (*rpc.Client, error) {
// 	conn, err := net.Dial("tcp", addr)
// 	if err != nil {
// 		return nil, fmt.Errorf("dial error: %s", err)
// 	}
// 	return rpc.NewClient(conn), nil
// }
