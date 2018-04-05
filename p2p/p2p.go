// Package p2p implements qri peer-to-peer communication.
package p2p

import (

	// gologging "gx/ipfs/QmQvJiADDe7JR4m968MwXobTCCzUqQkP87aRHe29MEBGHV/go-logging"
	// golog "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	golog "github.com/ipfs/go-log"
	identify "gx/ipfs/QmNh1kGFFdsPu79KNSaL4NUKUPb4Eiz4KHdMtFY6664RDp/go-libp2p/p2p/protocol/identify"
	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
)

var log = golog.Logger("qrip2p")

// QriProtocolID is the top level Protocol Identifier
const QriProtocolID = protocol.ID("/qri")

// QriServiceTag tags the type & version of the qri service
const QriServiceTag = "qri/0.2.1"

func init() {
	// golog.SetLogLevel("qrip2p", "debug")

	// ipfs core includes a client version. seems like a good idea.
	// TODO - understand where & how client versions are used
	identify.ClientVersion = QriServiceTag
}
