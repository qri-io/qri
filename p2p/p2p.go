package p2p

import (
	golog "github.com/ipfs/go-log"
	identify "github.com/libp2p/go-libp2p/p2p/protocol/identify"
	gologging "github.com/whyrusleeping/go-logging"
)

// Protocol Identifier
const ProtocolId = "/qri/0.0.1"

func init() {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	// ipfs core includes a client version. seems like a good idea.
	// TODO - understand where & how client versions are used
	identify.ClientVersion = "qri/0.0.1"
}
