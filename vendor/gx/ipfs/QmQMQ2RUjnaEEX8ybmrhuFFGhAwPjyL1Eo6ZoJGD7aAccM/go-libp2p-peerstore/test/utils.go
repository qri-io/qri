package testutil

import (
	"io"

	ci "gx/ipfs/QmNiCwBNA8MWDADTFVq1BonUEJbS2SvjAoNkZZrhEwcuUi/go-libp2p-crypto"
	peer "gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer"
	u "gx/ipfs/QmZuY8aV7zbNXVy6DyN9SmnuH3o9nG852F4aTiSBpts8d1/go-ipfs-util"
)

func RandPeerID() (peer.ID, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(u.NewTimeSeededRand(), buf); err != nil {
		return "", err
	}
	h := u.Hash(buf)
	return peer.ID(h), nil
}

func RandTestKeyPair(bits int) (ci.PrivKey, ci.PubKey, error) {
	return ci.GenerateKeyPairWithReader(ci.RSA, bits, u.NewTimeSeededRand())
}

func SeededTestKeyPair(seed int64) (ci.PrivKey, ci.PubKey, error) {
	return ci.GenerateKeyPairWithReader(ci.RSA, 512, u.NewSeededRand(seed))
}
