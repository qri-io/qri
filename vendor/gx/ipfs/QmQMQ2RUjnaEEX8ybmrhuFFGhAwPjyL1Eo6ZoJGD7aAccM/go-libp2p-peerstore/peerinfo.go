package peerstore

import (
	"encoding/json"

	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
	"gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer"
)

// PeerInfo is a small struct used to pass around a peer with
// a set of addresses (and later, keys?). This is not meant to be
// a complete view of the system, but rather to model updates to
// the peerstore. It is used by things like the routing system.
type PeerInfo struct {
	ID    peer.ID
	Addrs []ma.Multiaddr
}

func (pi *PeerInfo) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"peerID": pi.ID.Pretty(),
		"addrs":  pi.Addrs,
	}
}

func (pi *PeerInfo) MarshalJSON() ([]byte, error) {
	out := make(map[string]interface{})
	out["ID"] = pi.ID.Pretty()
	var addrs []string
	for _, a := range pi.Addrs {
		addrs = append(addrs, a.String())
	}
	out["Addrs"] = addrs
	return json.Marshal(out)
}

func (pi *PeerInfo) UnmarshalJSON(b []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	pid, err := peer.IDB58Decode(data["ID"].(string))
	if err != nil {
		return err
	}
	pi.ID = pid
	addrs, ok := data["Addrs"].([]interface{})
	if ok {
		for _, a := range addrs {
			pi.Addrs = append(pi.Addrs, ma.StringCast(a.(string)))
		}
	}
	return nil
}
