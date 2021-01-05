package lib

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

// ServeWebsocket creates a websocket that clients can connect to in order to
// get realtime events
func (inst *Instance) ServeWebsocket(ctx context.Context) {
	apiCfg := inst.cfg.API

	// Watch the filesystem. Events will be sent to websocket connections
	// TODO (b5) - watchfs constrcution shouldn't happen here
	watcher, err := watchfs.NewFilesysWatcher(ctx, inst.bus)
	if err != nil {
		log.Errorf("Watching filesystem error: %s", err)
		return
	}
	inst.watcher = watcher
	if err = inst.watcher.WatchAllFSIPaths(ctx, inst.repo); err != nil {
		log.Error(err)
	}

	addr, err := ma.NewMultiaddr(apiCfg.WebsocketAddress)
	if err != nil {
		log.Errorf("cannot start Websocket: error parsing Websocket address %s: %w", apiCfg.WebsocketAddress, err.Error())
	}

	mal, err := manet.Listen(addr)
	if err != nil {
		log.Infof("Websocket listen on address %d error: %w", apiCfg.WebsocketAddress, err.Error())
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

	inst.bus.Subscribe(handler,
		event.ETFSICreateLinkEvent,
		event.ETCreatedNewFile,
		event.ETModifiedFile,
		event.ETDeletedFile,
		event.ETRenamedFolder,
		event.ETRemovedFolder,

		event.ETRemoteClientPushVersionProgress,
		event.ETRemoteClientPushVersionCompleted,
		event.ETRemoteClientPushDatasetCompleted,
		event.ETRemoteClientPullVersionProgress,
		event.ETRemoteClientPullVersionCompleted,
		event.ETRemoteClientPullDatasetCompleted,
		event.ETRemoteClientRemoveDatasetCompleted,

		event.ETDatasetSaveStarted,
		event.ETDatasetSaveProgress,
		event.ETDatasetSaveCompleted,

		event.ETCronJobScheduled,
		event.ETCronJobUnscheduled,
		event.ETCronJobStarted,
		event.ETCronJobCompleted,
	)

	// Start http server for websocket.
	go func() {
		err = srv.Serve(l)
		if err != http.ErrServerClosed {
			log.Infof("failed to listen and serve: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err = srv.Shutdown(shutdownCtx); err != nil {
		log.Error(err)
	}
}
