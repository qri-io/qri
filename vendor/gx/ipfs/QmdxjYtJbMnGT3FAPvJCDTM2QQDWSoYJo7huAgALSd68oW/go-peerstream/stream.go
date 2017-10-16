package peerstream

import (
	"fmt"
	"time"

	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	smux "gx/ipfs/QmeZBgYBHvxMukGK5ojg28BCNLB9SeXqT7XXg6o7r2GbJy/go-stream-muxer"
)

// StreamHandler is a function which receives a Stream. It
// allows clients to set a function to receive newly created
// streams, and decide whether to continue adding them.
// It works sort of like a http.HandleFunc.
// Note: the StreamHandler is called sequentially, so spawn
// goroutines or pass the Stream. See EchoHandler.
type StreamHandler func(s *Stream)

// Stream is an io.{Read,Write,Close}r to a remote counterpart.
// It wraps a spdystream.Stream, and links it to a Conn and groups
type Stream struct {
	smuxStream smux.Stream

	conn     *Conn
	groups   groupSet
	protocol protocol.ID
}

func newStream(ss smux.Stream, c *Conn) *Stream {
	s := &Stream{
		conn:       c,
		smuxStream: ss,
		groups:     groupSet{m: make(map[Group]struct{})},
	}
	s.groups.AddSet(&c.groups) // inherit groups
	return s
}

// String returns a string representation of the Stream
func (s *Stream) String() string {
	f := "<peerstream.Stream %s <--> %s>"
	return fmt.Sprintf(f, s.conn.NetConn().LocalAddr(), s.conn.NetConn().RemoteAddr())
}

// SPDYStream returns the underlying *spdystream.Stream
func (s *Stream) Stream() smux.Stream {
	return s.smuxStream
}

// Conn returns the Conn associated with this Stream
func (s *Stream) Conn() *Conn {
	return s.conn
}

// Swarm returns the Swarm asociated with this Stream
func (s *Stream) Swarm() *Swarm {
	return s.conn.swarm
}

// Groups returns the Groups this Stream belongs to
func (s *Stream) Groups() []Group {
	return s.groups.Groups()
}

// InGroup returns whether this stream belongs to a Group
func (s *Stream) InGroup(g Group) bool {
	return s.groups.Has(g)
}

// AddGroup assigns given Group to Stream
func (s *Stream) AddGroup(g Group) {
	s.groups.Add(g)
}

func (s *Stream) Read(p []byte) (n int, err error) {
	return s.smuxStream.Read(p)
}

func (s *Stream) Write(p []byte) (n int, err error) {
	return s.smuxStream.Write(p)
}

func (s *Stream) Close() error {
	return s.conn.swarm.removeStream(s)
}

func (s *Stream) Protocol() protocol.ID {
	return s.protocol
}

func (s *Stream) SetProtocol(p protocol.ID) {
	s.protocol = p
}

func (s *Stream) SetDeadline(t time.Time) error {
	return s.smuxStream.SetDeadline(t)
}

func (s *Stream) SetReadDeadline(t time.Time) error {
	return s.smuxStream.SetReadDeadline(t)
}

func (s *Stream) SetWriteDeadline(t time.Time) error {
	return s.smuxStream.SetWriteDeadline(t)
}

// StreamsWithGroup narrows down a set of streams to those in given group.
func StreamsWithGroup(g Group, streams []*Stream) []*Stream {
	var out []*Stream
	for _, s := range streams {
		if s.InGroup(g) {
			out = append(out, s)
		}
	}
	return out
}
