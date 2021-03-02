package key

import (
	logger "github.com/ipfs/go-log"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

var log = logger.Logger("key")

// ID is a key identifier
// note that while this implementation is an alias for a peer.ID, this is not
// strictly a peerID.
type ID = peer.ID

// DecodeID parses an ID string
func DecodeID(s string) (ID, error) {
	pid, err := peer.Decode(s)
	if err != nil {
		return "", err
	}
	return ID(pid), nil
}
