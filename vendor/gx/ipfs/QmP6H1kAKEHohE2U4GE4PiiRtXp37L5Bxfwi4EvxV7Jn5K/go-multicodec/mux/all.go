package muxcodec

import (
	mc "gx/ipfs/QmP6H1kAKEHohE2U4GE4PiiRtXp37L5Bxfwi4EvxV7Jn5K/go-multicodec"
	cbor "gx/ipfs/QmP6H1kAKEHohE2U4GE4PiiRtXp37L5Bxfwi4EvxV7Jn5K/go-multicodec/cbor"
	json "gx/ipfs/QmP6H1kAKEHohE2U4GE4PiiRtXp37L5Bxfwi4EvxV7Jn5K/go-multicodec/json"
)

func StandardMux() *Multicodec {
	return MuxMulticodec([]mc.Multicodec{
		cbor.Multicodec(),
		json.Multicodec(false),
		json.Multicodec(true),
	}, SelectFirst)
}
