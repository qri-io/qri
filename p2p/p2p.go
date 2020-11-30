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
	QriServiceTag = fmt.Sprintf("qri/%s", version.Version)
	// ErrNotConnected is for a missing required network connection
	ErrNotConnected = fmt.Errorf("no p2p connection")
	// ErrQriProtocolNotSupported is returned when a connection can't be upgraded
	ErrQriProtocolNotSupported = fmt.Errorf("peer doesn't support the qri protocol")
	// ErrNoQriNode indicates a qri node doesn't exist
	ErrNoQriNode = fmt.Errorf("p2p: no qri node")
)

const (
	// depQriProtocolID is the top level Protocol Identifier
	// TODO (ramfox): soon to be removed - protocols now use semantic versioning
	depQriProtocolID = protocol.ID("/qri")
	// QriProtocolID is the protocol we use to determing if a node speaks qri
	// if it speaks the qri protocol, we assume it speaks any of the qri protocols
	// that we support
	QriProtocolID = protocol.ID("/qri/0.1.0")
	// default value to give qri peer connections in connmanager, one hunnit
	qriSupportValue = 100
	// qriSupportKey is the key we store the flag for qri support under in Peerstores and in ConnManager()
	qriSupportKey = "qri-support"
)

func init() {
	// golog.SetLogLevel("qrip2p", "debug")
	identify.ClientVersion = QriServiceTag
}
