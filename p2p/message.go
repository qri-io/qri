package p2p

import (
	"encoding/json"
	"math/rand"
	"time"

	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

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
// TODO - replace with UUIDs
func NewMessageID() string {
	b := make([]rune, 10)
	for i := range b {
		b[i] = alpharunes[rand.Intn(len(alpharunes))]
	}
	return string(b)
}
