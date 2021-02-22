package lib

import (
	"context"
	"net/http"

	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/fsi/watchfs"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const qriWebsocketProtocol = "qri-websocket"

// WebsocketHandler defines the handler interface
type WebsocketHandler interface {
	WSConnectionHandler(w http.ResponseWriter, r *http.Request)
}

// wsHandler is a concrete implementation of a websocket handler
// and serves to maintain the list of connections
type wsHandler struct {
	// Collect all websocket connections
	conns []*websocket.Conn
}

var _ WebsocketHandler = (*wsHandler)(nil)

// NewWebsocketHandler creates a new wsHandler instance that clients
// can connect to in order to get realtime events
func NewWebsocketHandler(ctx context.Context, inst *Instance) (WebsocketHandler, error) {
	ws := &wsHandler{
		conns: []*websocket.Conn{},
	}

	watcher, err := watchfs.NewFilesysWatcher(ctx, inst.bus)
	if err != nil {
		log.Errorf("Watching filesystem error: %s", err)
		return nil, err
	}
	inst.watcher = watcher
	if err = inst.watcher.WatchAllFSIPaths(ctx, inst.repo); err != nil {
		log.Error(err)
		return nil, err
	}
	inst.bus.SubscribeAll(ws.wsMessageHandler)

	return ws, nil
}

// WSConnectionHandler handles websocket upgrade requests and accepts the connection
func (h *wsHandler) WSConnectionHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:       []string{qriWebsocketProtocol},
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Debugf("Websocket accept error: %s", err)
		return
	}
	h.conns = append(h.conns, c)
}

func (h *wsHandler) wsMessageHandler(_ context.Context, e event.Event) error {
	ctx := context.Background()
	evt := map[string]interface{}{
		"type":      string(e.Type),
		"ts":        e.Timestamp,
		"sessionID": e.SessionID,
		"data":      e.Payload,
	}

	log.Debugf("sending event %q to %d websocket conns", e.Type, len(h.conns))
	for k, c := range h.conns {
		go func(k int, c *websocket.Conn) {
			err := wsjson.Write(ctx, c, evt)
			if err != nil {
				log.Errorf("connection %d: wsjson write error: %s", k, err)
			}
		}(k, c)
	}
	return nil
}
