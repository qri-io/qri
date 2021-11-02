package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/uuid"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/auth/token"
	"github.com/qri-io/qri/event"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const qriWebsocketProtocol = "qri-websocket"

var (
	errNotFound = fmt.Errorf("connection not found")

	log = golog.Logger("websocket")
)

// newID returns a new websocket connection ID
func newID() string {
	return uuid.New().String()
}

// setIDRand sets the random reader that NewID uses as a source of random bytes
// passing in nil will default to crypto.Rand. This can be used to make ID
// generation deterministic for tests. eg:
//    myString := "SomeRandomStringThatIsLong-SoYouCanCallItAsMuchAsNeeded..."
//    lib.SetIDRand(strings.NewReader(myString))
//    a := NewID()
//    lib.SetIDRand(strings.NewReader(myString))
//    b := NewID()
func setIDRand(r io.Reader) {
	uuid.SetRand(r)
}

// Handler defines the handler interface
type Handler interface {
	ConnectionHandler(w http.ResponseWriter, r *http.Request)
}

type connectionSet map[string]struct{}

// connections maintains the set of active websocket connections & associated
// connection metadata
type connections struct {
	conns         map[string]*conn
	connsLock     sync.Mutex
	keystore      key.Store
	subscriptions map[string]connectionSet
	subsLock      sync.Mutex
}

type conn struct {
	id        string
	profileID string
	conn      *websocket.Conn
}

var _ Handler = (*connections)(nil)

// NewHandler creates a new connections instance that clients
// can connect to in order to get realtime events
func NewHandler(ctx context.Context, bus event.Bus, keystore key.Store) (Handler, error) {
	ws := &connections{
		conns:         map[string]*conn{},
		connsLock:     sync.Mutex{},
		keystore:      keystore,
		subscriptions: map[string]connectionSet{},
		subsLock:      sync.Mutex{},
	}

	bus.SubscribeAll(ws.messageHandler)
	return ws, nil
}

// ConnectionHandler handles websocket upgrade requests and accepts the connection
func (h *connections) ConnectionHandler(w http.ResponseWriter, r *http.Request) {
	wsc, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:       []string{qriWebsocketProtocol},
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Debugf("Websocket accept error: %s", err)
		return
	}
	id := newID()
	c := &conn{
		id:   id,
		conn: wsc,
	}
	h.connsLock.Lock()
	defer h.connsLock.Unlock()
	h.conns[id] = c
	go h.read(id)
}

func (h *connections) messageHandler(_ context.Context, e event.Event) error {
	ctx := context.Background()
	evt := map[string]interface{}{
		"type":      string(e.Type),
		"ts":        e.Timestamp,
		"sessionID": e.SessionID,
		"data":      e.Payload,
	}

	profileIDString := e.ProfileID
	if profileIDString == "" {
		return nil
	}
	connIDs, err := h.getConnIDs(profileIDString)
	if err != nil {
		log.Errorf("profile %q: %w", profileIDString, err)
		return nil
	}

	for connID := range connIDs {
		c, err := h.getConn(connID)
		if err != nil {
			h.unsubscribeConn(profileIDString, connID)
			log.Errorf("connection %q, profile %q: %w", connID, profileIDString, err)
			return nil
		}
		log.Debugf("sending event %q to websocket conns %q", e.Type, profileIDString)
		if err := wsjson.Write(ctx, c.conn, evt); err != nil {
			log.Errorf("connection %q: wsjson write error: %s", profileIDString, err)
			return nil
		}
	}
	return nil
}

// getConn gets a *conn from the map of connections
func (h *connections) getConn(id string) (*conn, error) {
	h.connsLock.Lock()
	defer h.connsLock.Unlock()
	c, ok := h.conns[id]
	if !ok {
		return nil, errNotFound
	}
	return c, nil
}

// getConnID returns the connection ID associated with the given profile.ID string
func (h *connections) getConnIDs(profileID string) (connectionSet, error) {
	h.subsLock.Lock()
	defer h.subsLock.Unlock()
	ids, ok := h.subscriptions[profileID]
	if !ok {
		return nil, errNotFound
	}
	return ids, nil
}

// subscribeConn authenticates the given token and adds the connID to the map
// of "subscribed" connections
func (h *connections) subscribeConn(connID, tokenString string) error {
	ctx := context.TODO()
	tok, err := token.ParseAuthToken(ctx, tokenString, h.keystore)
	if err != nil {
		return err
	}

	claims, ok := tok.Claims.(*token.Claims)
	if !ok || claims.Subject == "" {
		return fmt.Errorf("cannot get profile.ID from token")
	}
	// TODO(b5): at this point we have a valid signature of a profileID string
	// but no proof that this profile is owned by the key that signed the
	// token. We either need ProfileID == KeyID, or we need a UCAN. we need to
	// check for those, ideally in a method within the profile package that
	// abstracts over profile & key agreement

	c, err := h.getConn(connID)
	if err != nil {
		return fmt.Errorf("connection %q: %w", connID, err)
	}
	c.profileID = claims.Subject

	h.subsLock.Lock()
	defer h.subsLock.Unlock()
	connIDs, ok := h.subscriptions[claims.Subject]
	if !ok || connIDs == nil {
		connIDs = connectionSet{}
	}
	connIDs[connID] = struct{}{}
	h.subscriptions[claims.Subject] = connIDs
	log.Debugw("subscribeConn", "id", connID)
	return nil
}

// unsubscribeConn remove the profileID and connID from the map of "subscribed"
// connections
func (h *connections) unsubscribeConn(profileID, connID string) {
	connIDs, err := h.getConnIDs(profileID)
	if err != nil {
		return
	}

	for cid := range connIDs {
		if connID == "" || cid == connID {
			c, err := h.getConn(cid)
			if err != nil {
				continue
			}
			c.profileID = ""
		}
	}

	h.subsLock.Lock()
	defer h.subsLock.Unlock()
	if connID == "" {
		delete(h.subscriptions, profileID)
	} else {
		if _, ok := h.subscriptions[profileID]; ok {
			delete(h.subscriptions[profileID], connID)
		}
	}
}

// removeConn removes the conn from the map of connections and subscriptions
// closing the connection if needed
func (h *connections) removeConn(id string) {
	c, err := h.getConn(id)
	if err != nil {
		return
	}
	defer func() {
		c.conn.Close(websocket.StatusNormalClosure, "pruning connection")
	}()
	if c.profileID != "" {
		h.unsubscribeConn(c.profileID, id)
	}
	h.connsLock.Lock()
	defer h.connsLock.Unlock()
	delete(h.conns, id)
}

// read listens to the given connection, handling any messages that come through
// stops listening if it encounters any error
func (h *connections) read(id string) error {
	msg := &message{}

	c, err := h.getConn(id)
	if err != nil {
		return fmt.Errorf("connection %q: %w", id, err)
	}
	ctx := context.Background()
	for {
		err = wsjson.Read(ctx, c.conn, msg)
		if err != nil {
			// all websocket methods that return w/ failure are closed
			// we must prune the closed connection
			h.removeConn(id)
			return err
		}
		h.handleMessage(ctx, c, msg)
	}
}

// handleMessage handles each message based on msgType
func (h *connections) handleMessage(ctx context.Context, c *conn, msg *message) {
	switch msg.Type {
	case subscribeRequest:
		subMsg := &subscribeMessage{}
		if err := json.Unmarshal(msg.Payload, subMsg); err != nil {
			log.Debugw("websocket unmarshal", "error", err, "connection id", c.id, "msg", msg)
			h.write(ctx, c, &message{Type: subscribeFailure, Error: err})
			return
		}
		if err := h.subscribeConn(c.id, subMsg.Token); err != nil {
			log.Debugw("subscribeConn", "error", err, "connection id", c.id, "msg", msg)
			h.write(ctx, c, &message{Type: subscribeFailure, Error: err})
			return
		}
		h.write(ctx, c, &message{Type: subscribeSuccess})
	case unsubscribeRequest:
		h.unsubscribeConn(c.profileID, c.id)
	default:
		log.Debug("unknown message type over websocket %s: %q", c.id, msg.Type)
	}
}

// write sends a json message over the connection
func (h *connections) write(ctx context.Context, c *conn, msg *message) {
	log.Debugf("sending message %q to websocket conns %q", msg.Type, c.id)
	if err := wsjson.Write(ctx, c.conn, msg); err != nil {
		log.Errorf("connection %q: wsjson write error: %s", c.id, err)
		// the connection will close if there is any `write` error
		// we must remove it from our own stores, so as not to hold
		// onto any dead connections
		h.removeConn(c.id)
	}
}

// msgType is the type of message that we receive on the
type msgType string

const (
	// subscribeRequest indicates the connection is trying to become
	// an authenticated connection
	// payload is a `subscribeMessage`
	subscribeRequest = msgType("subscribe:request")
	// subscribeSuccess indicates that the connection successfully
	// upgraded to an authenticated connection
	// payload is nil
	subscribeSuccess = msgType("subscribe:success")
	// subscribeFailure indicates that the connection did not
	// upgrade to an authenticated connection
	// payload is nil
	subscribeFailure = msgType("subscribe:failure")
	// unsubscribeRequest indicates the connection no longer wants
	// to be authenticated
	// payload is nil
	unsubscribeRequest = msgType("unsubscribe:request")
)

// message is the expected structure of an incoming websocket message
type message struct {
	Type    msgType         `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Error   error           `json:"error"`
}

// subscribeMessage is the expected structure of an incoming "subscribe"
// message
type subscribeMessage struct {
	Token string `json:"token"`
}
