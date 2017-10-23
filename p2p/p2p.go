package p2p

import (
	golog "github.com/ipfs/go-log"
	gologging "github.com/whyrusleeping/go-logging"

	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	identify "gx/ipfs/QmefgzMbKZYsmHFkLqxgaTBG9ypeEjrdWRD5WXH4j1cWDL/go-libp2p/p2p/protocol/identify"
)

// Protocol Identifier
const QriProtocolId = protocol.ID("/qri")
const QriServiceTag = "qri/0.0.1"

func init() {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	// ipfs core includes a client version. seems like a good idea.
	// TODO - understand where & how client versions are used
	identify.ClientVersion = QriServiceTag
}
