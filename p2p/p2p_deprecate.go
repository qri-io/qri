package p2p

// TODO (ramfox): all the code in this file is marked for deprecation
// it is what we are currently relying on to keep up support for the
// `depQriProtocolID`

// * `depQriStreamHandler`
// * old profile handler, which is needed to allow nodes that use the
// old qri protocol to know about this nodes existance
// * `Message` which was used when the libp2p pattern was to use
// one protocol and have a way to route messages to their specific
// handlers
// * `receiveMessage` and `sendMessage`, which are methods on `WrapStream`
// that rely on `Message`

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	net "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qri/event"
)

// depQriStreamHandler is the handler we register with the `depQriProtocolID`
// it will respond saying informing the sender that this node does not support
// the old protocol and to please update
// TODO (ramfox): once the `depQriProtocolID` is removed, we can remove this as well
func (n *QriNode) depQriStreamHandler(s net.Stream) {
	// defer s.Close()
	ws := WrapStream(s)
	msg, err := ws.receiveMessage()
	if err != nil {
		if err.Error() == "EOF" {
			return
		}
		log.Debugf("error receiving message: %s", err.Error())
		return
	}

	// note: we are still handling profile requests and until qri 0.10.0, since the
	// old qriProtocol asks for to ask for a profile when determining its qri peers
	// we are still handling DatasetsList because lib.Peers is currently relying on it
	switch msg.Type {
	case MtProfile:
		n.handleProfile(ws, msg)
		break
	case MtDatasets:
		n.handleDatasetsList(ws, msg)
		break
	default:
		res := NewMessage(n.host.ID(), msg.Type, []byte("The Qri peer you are trying to communicate with no longer supports the deprecated qri protocol that this stream is using. Please update to the latest version of Qri!"))
		ws.sendMessage(res)
	}

	n.pub.Publish(context.Background(), event.ETP2PMessageReceived, msg)
	go n.writeToReceivers(msg)

	ws.stream.Close()
}

// MtProfile is a peer info message
const MtProfile = MsgType("profile")

func (n *QriNode) handleProfile(ws *WrappedStream, msg Message) (hangup bool) {
	switch msg.Header("phase") {
	case "request":

		data, err := n.profileBytes()
		if err != nil {
			log.Debug(err.Error())
			return
		}

		if err := ws.sendMessage(msg.Update(data)); err != nil {
			log.Debugf("error sending peer info message: %s", err.Error())
		}
	}
	return
}

func (n *QriNode) profileBytes() ([]byte, error) {
	p, err := n.Repo.Profile()
	if err != nil {
		log.Debugf("error getting repo profile: %s\n", err.Error())
		return nil, err
	}
	pod, err := p.Encode()
	if err != nil {
		log.Debugf("error encoding repo profile: %s\n", err.Error())
		return nil, err
	}

	return json.Marshal(pod)
}

func (n *QriNode) writeToReceivers(msg Message) {
	n.receiversMu.Lock()
	defer n.receiversMu.Unlock()
	for _, r := range n.receivers {
		r <- msg
	}
}

// MsgType indicates the type of message being sent
type MsgType string

// String implements the Stringer interface for MsgType
func (mt MsgType) String() string {
	return string(mt)
}

// Message is a serializable/encodable object that we send & receive on a Stream.
type Message struct {
	Type     MsgType
	ID       string
	Created  time.Time
	Deadline time.Time
	// peer that originated this message
	Initiator peer.ID
	// Headers proxies the concept of HTTP headers, but with no
	// mandatory fields. It's intended to be small & simple on purpose
	// In the future we can upgrade this to map[string]interface{} while keeping
	// backward compatibility
	Headers map[string]string
	// Body carries the payload of a message, if any
	Body []byte
	// provider is who sent this message
	// not transmitted over the wire, but
	// instead populated by WrapStream
	provider peer.ID
}

// Update returns a new message with an updated body
func (m Message) Update(body []byte) Message {
	return Message{
		Type:      m.Type,
		ID:        m.ID,
		Created:   m.Created,
		Deadline:  m.Deadline,
		Initiator: m.Initiator,
		Body:      body,
	}
}

// UpdateJSON updates a messages by JSON-encoding a body
func (m Message) UpdateJSON(body interface{}) (Message, error) {
	data, err := json.Marshal(body)
	return m.Update(data), err
}

// NewMessage creates a message. provided initiator should always be the peerID
// of the local node
func NewMessage(initiator peer.ID, t MsgType, body []byte) Message {
	return Message{
		ID:        NewMessageID(),
		Initiator: initiator,
		Created:   time.Now(),
		Deadline:  time.Now().Add(time.Minute * 2),
		Type:      t,
		Headers:   map[string]string{},
		Body:      body,
	}
}

// WithHeaders adds a sequence of key,value,key,value as headers
func (m Message) WithHeaders(keyval ...string) Message {
	headers := map[string]string{}
	for i := 0; i < len(keyval)-1; i = i + 2 {
		headers[keyval[i]] = keyval[i+1]
	}
	return Message{
		ID:        m.ID,
		Initiator: m.Initiator,
		Type:      m.Type,
		Headers:   headers,
		Body:      m.Body,
	}
}

// Header gets a header value for a given key
func (m Message) Header(key string) (value string) {
	if m.Headers == nil {
		return ""
	}
	return m.Headers[key]
}

// NewJSONBodyMessage is a convenience wrapper for json-encoding a message
func NewJSONBodyMessage(initiator peer.ID, t MsgType, body interface{}) (Message, error) {
	data, err := json.Marshal(body)
	return NewMessage(initiator, t, data), err
}

var alpharunes = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// NewMessageID generates a random message identifier
func NewMessageID() string {
	b := make([]rune, 10)
	for i := range b {
		b[i] = alpharunes[rand.Intn(len(alpharunes))]
	}
	return string(b)
}

// Wrapped stream dep methods
// receiveMessage reads and decodes a message from the stream
func (ws *WrappedStream) receiveMessage() (msg Message, err error) {
	err = ws.dec.Decode(&msg)
	msg.provider = ws.stream.Conn().RemotePeer()
	log.Debugf("%s '%s' <- %s", ws.stream.Conn().LocalPeer(), msg.Type, ws.stream.Conn().RemotePeer())
	return
}

// sendMessage encodes and writes a message to the stream
func (ws *WrappedStream) sendMessage(msg Message) error {
	err := ws.enc.Encode(&msg)
	// Because output is buffered with bufio, we need to flush!
	ws.w.Flush()
	log.Debugf("%s '%s' -> %s", ws.stream.Conn().LocalPeer(), msg.Type, ws.stream.Conn().RemotePeer())
	return err
}
