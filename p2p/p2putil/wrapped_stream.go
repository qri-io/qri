package p2putil

import (
	"bufio"

	net "github.com/libp2p/go-libp2p-core/network"
	json "github.com/qri-io/dag/dsync/p2putil/json"
	multicodec "github.com/qri-io/dag/dsync/p2putil/multicodec_old"
)

// HandlerFunc is the signature of a function that can handle p2p messages
type HandlerFunc func(ws *WrappedStream, msg Message) (hangup bool)

// WrappedStream wraps a libp2p stream. We encode/decode whenever we
// write/read from a stream, so we can just carry the encoders
// and bufios with us
type WrappedStream struct {
	stream net.Stream
	Enc    multicodec.Encoder
	Dec    multicodec.Decoder
	W      *bufio.Writer
	R      *bufio.Reader
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
		R:      reader,
		W:      writer,
		Enc:    enc,
		Dec:    dec,
	}
}

// ReceiveMessage reads and decodes a message from the stream
func (ws *WrappedStream) ReceiveMessage() (msg Message, err error) {
	err = ws.Dec.Decode(&msg)
	msg.provider = ws.stream.Conn().RemotePeer()
	// log.Debugf("%s '%s' <- %s", ws.stream.Conn().LocalPeer(), msg.Type, ws.stream.Conn().RemotePeer())
	return
}

// SendMessage encodes and writes a message to the stream
func (ws *WrappedStream) SendMessage(msg Message) error {
	err := ws.Enc.Encode(&msg)
	// Because output is buffered with bufio, we need to flush!
	ws.W.Flush()
	// log.Debugf("%s '%s' -> %s", ws.stream.Conn().LocalPeer(), msg.Type, ws.stream.Conn().RemotePeer())
	return err
}

// Stream exposes the underlying libp2p net.Stream
func (ws *WrappedStream) Stream() net.Stream {
	return ws.stream
}

// Close closes the stream connection
func (ws *WrappedStream) Close() error {
	return ws.stream.Close()
}
