package p2p

import (
	"bufio"

	net "github.com/libp2p/go-libp2p-core/network"
	multicodec "github.com/multiformats/go-multicodec"
	json "github.com/multiformats/go-multicodec/json"
)

// HandlerFunc is the signature of a function that can handle p2p messages
type HandlerFunc func(ws *WrappedStream, msg Message) (hangup bool)

// WrappedStream wraps a libp2p stream. We encode/decode whenever we
// write/read from a stream, so we can just carry the encoders
// and bufios with us
type WrappedStream struct {
	stream net.Stream
	enc    multicodec.Encoder
	dec    multicodec.Decoder
	w      *bufio.Writer
	r      *bufio.Reader
}

// WrapStream takes a stream and complements it with r/w bufios and
// decoder/encoder. In order to write raw data to the stream we can use
// wrap.w.Write(). To encode something into it we can wrap.enc.Encode().
// Finally, we should wrap.w.Flush() to actually send the data. Handling
// incoming data works similarly with wrap.r.Read() for raw-reading and
// wrap.dec.Decode() to decode.
func WrapStream(s net.Stream) *WrappedStream {
	reader := bufio.NewReader(s)
	writer := bufio.NewWriter(s)
	// This is where we pick our specific multicodec. In order to change the
	// codec, we only need to change this place.
	// See https://godoc.org/github.com/multiformats/go-multicodec/json
	dec := json.Multicodec(false).Decoder(reader)
	enc := json.Multicodec(false).Encoder(writer)
	return &WrappedStream{
		stream: s,
		r:      reader,
		w:      writer,
		enc:    enc,
		dec:    dec,
	}
}

// TODO (ramfox): currently, each protocol (except for the one marked for deprecation)
// has its own `receiveMessage` and `sendMessage` functions, that ensure the
// messages get encoded and decoded to the structures they need. We should figure out
// how to generalize so that each protocol can pass in the way they want to
// be able to encode and decode the stream to suit their data needs
// // receiveMessage reads and decodes a message from the stream
// func (ws *WrappedStream) receiveMessage() (msg Message, err error) {
// 	err = ws.dec.Decode(&msg)
// 	msg.provider = ws.stream.Conn().RemotePeer()
// 	log.Debugf("%s '%s' <- %s", ws.stream.Conn().LocalPeer(), msg.Type, ws.stream.Conn().RemotePeer())
// 	return
// }

// // sendMessage encodes and writes a message to the stream
// func (ws *WrappedStream) sendMessage(msg Message) error {
// 	err := ws.enc.Encode(&msg)
// 	// Because output is buffered with bufio, we need to flush!
// 	ws.w.Flush()
// 	log.Debugf("%s '%s' -> %s", ws.stream.Conn().LocalPeer(), msg.Type, ws.stream.Conn().RemotePeer())
// 	return err
// }
