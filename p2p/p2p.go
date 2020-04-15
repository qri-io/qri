// Package p2p implements qri peer-to-peer communication.
package p2p

import (
	"fmt"

	golog "github.com/ipfs/go-log"
	protocol "github.com/libp2p/go-libp2p-core/protocol"
	identify "github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/qri-io/qri/version"
)

var (
	log = golog.Logger("qrip2p")
	// QriServiceTag marks the type & version of the qri service
	QriServiceTag = fmt.Sprintf("qri/%s", version.String)
	// ErrNotConnected is for a missing required network connection
	ErrNotConnected = fmt.Errorf("no p2p connection")
	// ErrQriProtocolNotSupported is returned when a connection can't be upgraded
	ErrQriProtocolNotSupported = fmt.Errorf("peer doesn't support the qri protocol")
	// ErrNoQriNode indicates a qri node doesn't exist
	ErrNoQriNode = fmt.Errorf("p2p: no qri node")
)

const (
	// QriProtocolID is the top level Protocol Identifier
	QriProtocolID = protocol.ID("/qri")
	// default value to give qri peer connections in connmanager, one hunnit
	qriSupportValue = 100
	// qriSupportKey is the key we store the flag for qri support under in Peerstores and in ConnManager()
	qriSupportKey = "qri-support"
)

func init() {
	// golog.SetLogLevel("qrip2p", "debug")
	identify.ClientVersion = QriServiceTag
}
