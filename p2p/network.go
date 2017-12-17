package p2p

import (
	"fmt"
	"net"
)

// LocalOpenPort looks for the first open port, starting at start,
// incrementing by 1 port num until a number is found
// network must be one of: "tcp", "tcp4", "tcp6", "unix" or "unixpacket"
func LocalOpenPort(network string, start int) (int, error) {
	if start > 100000 {
		return 0, fmt.Errorf("couldn't find an open port to bind to")
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", start))
	if err != nil {
		return LocalOpenPort(network, start+1)
	}
	ln.Close()

	return start, nil
}
