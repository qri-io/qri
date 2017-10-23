package basemux

import (
	mc "gx/ipfs/QmP6H1kAKEHohE2U4GE4PiiRtXp37L5Bxfwi4EvxV7Jn5K/go-multicodec"
	mux "gx/ipfs/QmP6H1kAKEHohE2U4GE4PiiRtXp37L5Bxfwi4EvxV7Jn5K/go-multicodec/mux"

	b64 "gx/ipfs/QmP6H1kAKEHohE2U4GE4PiiRtXp37L5Bxfwi4EvxV7Jn5K/go-multicodec/base/b64"
	bin "gx/ipfs/QmP6H1kAKEHohE2U4GE4PiiRtXp37L5Bxfwi4EvxV7Jn5K/go-multicodec/base/bin"
	hex "gx/ipfs/QmP6H1kAKEHohE2U4GE4PiiRtXp37L5Bxfwi4EvxV7Jn5K/go-multicodec/base/hex"
)

func AllBasesMux() *mux.Multicodec {
	m := mux.MuxMulticodec([]mc.Multicodec{
		hex.Multicodec(),
		b64.Multicodec(),
		bin.Multicodec(),
	}, mux.SelectFirst)
	m.Wrap = false
	return m
}
