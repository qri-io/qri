package addrutil

import (
	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
	mafmt "gx/ipfs/QmYjJnSTfXWhYL2cV1xFphPqjqowJqH7ZKLA1As8QrPHbn/mafmt"
)

// SubtractFilter returns a filter func that filters all of the given addresses
func SubtractFilter(addrs ...ma.Multiaddr) func(ma.Multiaddr) bool {
	addrmap := make(map[string]bool)
	for _, a := range addrs {
		addrmap[string(a.Bytes())] = true
	}

	return func(a ma.Multiaddr) bool {
		return !addrmap[string(a.Bytes())]
	}
}

// IsFDCostlyTransport returns true for transports that require a new file
// descriptor per connection created
func IsFDCostlyTransport(a ma.Multiaddr) bool {
	return mafmt.TCP.Matches(a)
}

// FilterNeg returns a negated version of the passed in filter
func FilterNeg(f func(ma.Multiaddr) bool) func(ma.Multiaddr) bool {
	return func(a ma.Multiaddr) bool {
		return !f(a)
	}
}
