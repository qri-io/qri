// Package p2p implements qri peer-to-peer communication.
package p2p

import (
	"fmt"

	golog "github.com/ipfs/go-log"

	identify "gx/ipfs/QmY51bqSM5XgxQZqsBrQcRkKTnCb8EKpJpR9K6Qax7Njco/go-libp2p/p2p/protocol/identify"
	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
)

var log = golog.Logger("qrip2p")

const (
	// QriProtocolID is the top level Protocol Identifier
	QriProtocolID = protocol.ID("/qri")
	// QriServiceTag tags the type & version of the qri service
	QriServiceTag = "qri/0.5.0"
	// tag qri service uses in host connection Manager
	qriConnManagerTag = "qri"
	// default value to give qri peer connections in connmanager
	qriConnManagerValue = 6
)

func init() {
	// golog.SetLogLevel("qrip2p", "debug")

	// ipfs core includes a client version. seems like a good idea.
	// TODO - understand where & how client versions are used
	identify.ClientVersion = QriServiceTag
}

// ErrNotConnected is for a missing required network connection
var ErrNotConnected = fmt.Errorf("no p2p connection")
