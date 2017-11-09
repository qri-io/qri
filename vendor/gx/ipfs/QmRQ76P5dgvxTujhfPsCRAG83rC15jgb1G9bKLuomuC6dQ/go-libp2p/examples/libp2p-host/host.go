package main

import (
	"context"
	"crypto/rand"
	"fmt"

	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	ma "github.com/multiformats/go-multiaddr"
)

func main() {
	// Generate an identity keypair using go's cryptographic randomness source
	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		panic(err)
	}

	// A peers ID is the hash of its public key
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		panic(err)
	}

	// We've created the identity, now we need to store it.
	// A peerstore holds information about peers, including your own
	ps := pstore.NewPeerstore()
	ps.AddPrivKey(pid, priv)
	ps.AddPubKey(pid, pub)

	maddr, err := ma.NewMultiaddr("/ip4/0.0.0.0/tcp/9000")
	if err != nil {
		panic(err)
	}

	// Make a context to govern the lifespan of the swarm
	ctx := context.Background()

	// Put all this together
	netw, err := swarm.NewNetwork(ctx, []ma.Multiaddr{maddr}, pid, ps, nil)
	if err != nil {
		panic(err)
	}

	myhost := bhost.New(netw)
	fmt.Printf("Hello World, my hosts ID is %s\n", myhost.ID())
}
