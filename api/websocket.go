package api

import (
	"context"
	"fmt"

	"net/http"
	"time"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/watchfs"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const (
	qriWebsocketProtocol = "qri-websocket"
)

// TODO(dlong): This file has a tight coupling between Websocket and Watchfs that makes sense
// for now, as they're two pieces working together on the same task, but will start to make
// less sense once more Websocket messages are being delivered, and as the event.Bus is used
// more places. Reconsider in the future how to better integrate these two pieces.

// ServeWebsocket creates a websocket that clients can connect to in order to get realtime events
func (s Server) ServeWebsocket(ctx context.Context) {
	c := s.Config().API
	// Watch the filesystem. Events will be sent to websocket connections.
	node := s.Node()
	fsmessages, err := s.startFilesysWatcher(ctx, node)
	if err != nil {
		log.Infof("Watching filesystem error: %s", err)
		return
	}

	go func() {
		maAddress := c.WebsocketAddress
		addr, err := ma.NewMultiaddr(maAddress)
		if err != nil {
			log.Errorf("cannot start Websocket: error parsing Websocket address %s: %w", maAddress, err.Error())
		}

		mal, err := manet.Listen(addr)
		if err != nil {
			log.Infof("Websocket listen on address %d error: %w", c.WebsocketAddress, err.Error())
			return
		}
		l := manet.NetListener(mal)
		defer l.Close()

		// Collect all websocket connections. Should only be one at a time, but that may
		// change in the future.
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

		// Subscribe to FSI link creation events, which will affect filesystem watching
		// TODO(dlong): A good example of tight coupling causing an issue: The Websocket
		// implementation doesn't need to know about these events, but the FilesystemWatcher
		// does. Ideally, this Subscribe call would happen along with the latter, not the former.
		busEvents := s.Instance.Bus().Subscribe(event.ETFSICreateLinkEvent)

		known := component.GetKnownFilenames()

		// Filesystem events are forwarded to the websocket. In the future, this may be
		// expanded to handle other types of events, such as SaveDatasetProgressEvent,
		// and DiffProgressEvent, but this is fine for now.
		go func() {
			for {
				select {
				case e := <-busEvents:
					log.Debugf("bus event: %s\n", e)
					if fce, ok := e.Payload.(event.FSICreateLinkEvent); ok {
						s.Instance.Watcher.Add(watchfs.EventPath{
							Path:     fce.FSIPath,
							Username: fce.Username,
							Dsname:   fce.Dsname,
						})
					}
				case fse := <-fsmessages:
					if s.filterEvent(fse, known) {
						log.Debugf("filesys event: %s\n", fse)
						for k, c := range connections {
							err = wsjson.Write(ctx, c, fse)
							if err != nil {
								log.Errorf("connection %d: wsjson write error: %s", k, err)
							}
						}
					}
				}
			}
		}()

		// TODO(dlong): Move to SummaryString
		fmt.Printf("Listening for websocket connection at %s\n", l.Addr().String())

		// Start http server for websocket.
		err = srv.Serve(l)
		if err != http.ErrServerClosed {
			log.Infof("failed to listen and serve: %v", err)
		}
	}()
}

func (s Server) startFilesysWatcher(ctx context.Context, node *p2p.QriNode) (chan watchfs.FilesysEvent, error) {
	refs, err := node.Repo.References(0, 100)
	if err != nil {
		return nil, err
	}
	// Extract fsi paths for all working directories.
	paths := make([]watchfs.EventPath, 0, len(refs))
	for _, ref := range refs {
		if ref.FSIPath != "" {
			paths = append(paths, watchfs.EventPath{
				Path:     ref.FSIPath,
				Username: ref.Peername,
				Dsname:   ref.Name,
			})
		}
	}
	// Watch those paths.
	// TODO(dlong): When datasets are removed or renamed update the watchlist.
	s.Instance.Watcher = watchfs.NewFilesysWatcher(ctx, s.Instance.Bus())
	fsmessages := s.Instance.Watcher.Begin(paths)
	return fsmessages, nil
}

func (s Server) filterEvent(event watchfs.FilesysEvent, knownFilenames map[string][]string) bool {
	return component.IsKnownFilename(event.Source, knownFilenames)
}
