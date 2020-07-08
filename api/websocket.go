package api

import (
	"context"

	"net/http"
	"time"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/watchfs"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const qriWebsocketProtocol = "qri-websocket"

// ServeWebsocket creates a websocket that clients can connect to in order to get realtime events
func (s Server) ServeWebsocket(ctx context.Context) {
	cfg := s.Config().API

	// Watch the filesystem. Events will be sent to websocket connections.
	watcher, err := watchfs.NewFilesysWatcher(ctx, s.Instance.Bus())
	if err != nil {
		log.Errorf("Watching filesystem error: %s", err)
		return
	}
	s.Instance.Watcher = watcher
	if err = s.Instance.Watcher.WatchAllFSIPaths(ctx, s.Repo()); err != nil {
		log.Error(err)
	}

	addr, err := ma.NewMultiaddr(cfg.WebsocketAddress)
	if err != nil {
		log.Errorf("cannot start Websocket: error parsing Websocket address %s: %w", cfg.WebsocketAddress, err.Error())
	}

	mal, err := manet.Listen(addr)
	if err != nil {
		log.Infof("Websocket listen on address %d error: %w", cfg.WebsocketAddress, err.Error())
		return
	}
	l := manet.NetListener(mal)
	defer l.Close()

	// Collect all websocket connections
	connections := []*websocket.Conn{}
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
				Subprotocols:       []string{qriWebsocketProtocol},
				InsecureSkipVerify: true,
			})
			if err != nil {
				log.Debugf("Websocket accept error: %s", err)
				return
			}
			connections = append(connections, c)
		}),
		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
	}
	defer srv.Close()

	handler := func(_ context.Context, t event.Type, payload interface{}) error {
		ctx := context.Background()
		evt := map[string]interface{}{
			"type": string(t),
			"data": payload,
		}

		log.Debugf("sending event %q to %d websocket conns", t, len(connections))
		for k, c := range connections {
			go func(k int, c *websocket.Conn) {
				err := wsjson.Write(ctx, c, evt)
				if err != nil {
					log.Errorf("connection %d: wsjson write error: %s", k, err)
				}
			}(k, c)
		}
		return nil
	}

	s.Instance.Bus().Subscribe(handler,
		event.ETFSICreateLinkEvent,
		event.ETCreatedNewFile,
		event.ETModifiedFile,
		event.ETDeletedFile,
		event.ETRenamedFolder,
		event.ETRemovedFolder,
	)

	// Start http server for websocket.
	err = srv.Serve(l)
	if err != http.ErrServerClosed {
		log.Infof("failed to listen and serve: %v", err)
	}
}
