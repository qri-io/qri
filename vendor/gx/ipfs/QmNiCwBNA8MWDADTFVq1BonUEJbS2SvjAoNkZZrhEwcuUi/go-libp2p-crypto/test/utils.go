package testutil

import (
	"math/rand"
	"time"

	ci "gx/ipfs/QmNiCwBNA8MWDADTFVq1BonUEJbS2SvjAoNkZZrhEwcuUi/go-libp2p-crypto"
)

func RandTestKeyPair(bits int) (ci.PrivKey, ci.PubKey, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return ci.GenerateKeyPairWithReader(ci.RSA, bits, r)
}

func SeededTestKeyPair(seed int64) (ci.PrivKey, ci.PubKey, error) {
	r := rand.New(rand.NewSource(seed))
	return ci.GenerateKeyPairWithReader(ci.RSA, 512, r)
}
