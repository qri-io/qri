// Package api implements a JSON-API for interacting with a qri node
package api

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	stdlog "log"
	"net"
	"net/http"
	"net/rpc"
	"time"

	"github.com/datatogether/api/apiutil"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dag"
	"github.com/qri-io/dag/dsync"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"

	"gx/ipfs/QmUJYo4etAQqFfSS2rarFAE97eNGB8ej64YkRT2SmsYD4r/go-ipfs/core/coreapi"
	coreiface "gx/ipfs/QmUJYo4etAQqFfSS2rarFAE97eNGB8ej64YkRT2SmsYD4r/go-ipfs/core/coreapi/interface"
)

var log = golog.Logger("qriapi")

// LocalHostIP is the IP address for localhost
const LocalHostIP = "127.0.0.1"

func init() {
	// We don't use the log package, and the net/rpc package spits out some complaints b/c
	// a few methods don't conform to the proper signature (comment this out & run 'qri connect' to see errors)
	// so we're disabling the log package for now. This is potentially very stupid.
	// TODO (b5): remove dep on net/rpc package entirely
	stdlog.SetOutput(ioutil.Discard)

	golog.SetLogLevel("qriapi", "info")
}

// Server wraps a qri p2p node, providing traditional access via http
// Create one with New, start it up with Serve
type Server struct {
	// configuration options
	cfg     *config.Config
	qriNode *p2p.QriNode
}

// New creates a new qri server from a p2p node & configuration
func New(node *p2p.QriNode, cfg *config.Config) (s *Server) {
	return &Server{
		qriNode: node,
		cfg:     cfg,
	}
}

// Serve starts the server. It will block while the server is running
func (s *Server) Serve() (err error) {
	if err = s.qriNode.GoOnline(); err != nil {
		fmt.Println("serving error", s.cfg.P2P.Enabled)
		return
	}

	server := &http.Server{}
	mux := NewServerRoutes(s)
	server.Handler = mux

	go s.ServeRPC()
	go s.ServeWebapp()

	if node, err := s.qriNode.IPFSNode(); err == nil {
		if pinner, ok := s.qriNode.Repo.Store().(cafs.Pinner); ok {

			go func() {
				if _, err := lib.CheckVersion(context.Background(), node.Namesys, lib.PrevIPNSName, lib.LastPubVerHash); err == lib.ErrUpdateRequired {
					log.Info("This version of qri is out of date, please refer to https://github.com/qri-io/qri/releases/latest for more info")
				} else if err != nil {
					log.Infof("error checking for software update: %s", err.Error())
				}
			}()

			go func() {
				// TODO - this is breaking encapsulation pretty hard. Should probs move this stuff into lib
				if lib.Config != nil && lib.Config.Render != nil && lib.Config.Render.TemplateUpdateAddress != "" {
					if latest, err := lib.CheckVersion(context.Background(), node.Namesys, lib.Config.Render.TemplateUpdateAddress, lib.Config.Render.DefaultTemplateHash); err == lib.ErrUpdateRequired {
						err := pinner.Pin(latest, true)
						if err != nil {
							log.Debug("error pinning template hash: %s", err.Error())
							return
						}
						if err := lib.Config.Set("Render.DefaultTemplateHash", latest); err != nil {
							log.Debug("error setting latest hash: %s", err.Error())
							return
						}
						if err := lib.SaveConfig(); err != nil {
							log.Debug("error saving config hash: %s", err.Error())
							return
						}
						log.Info("auto-updated template hash: %s", latest)
					}
				}
			}()

		}
	}

	info := "\n📡  Success! You are now connected to the d.web. Here's your connection details:\n"
	info += s.cfg.SummaryString()
	info += "IPFS Addresses:"
	for _, a := range s.qriNode.EncapsulatedAddresses() {
		info = fmt.Sprintf("%s\n  %s", info, a.String())
	}
	info += "\n\n"

	s.qriNode.LocalStreams.Print(info)

	if s.cfg.API.DisconnectAfter != 0 {
		log.Infof("disconnecting after %d seconds", s.cfg.API.DisconnectAfter)
		go func(s *http.Server, t int) {
			<-time.After(time.Second * time.Duration(t))
			log.Infof("disconnecting")
			s.Close()
		}(server, s.cfg.API.DisconnectAfter)
	}

	// http.ListenAndServe will not return unless there's an error
	return StartServer(s.cfg.API, server)
}

// ServeRPC checks for a configured RPC port, and registers a listner if so
func (s *Server) ServeRPC() {
	if !s.cfg.RPC.Enabled || s.cfg.RPC.Port == 0 {
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", LocalHostIP, s.cfg.RPC.Port))
	if err != nil {
		log.Infof("RPC listen on port %d error: %s", s.cfg.RPC.Port, err)
		return
	}

	for _, rcvr := range lib.Receivers(s.qriNode) {
		if err := rpc.Register(rcvr); err != nil {
			log.Infof("error registering RPC receiver %s: %s", rcvr.CoreRequestsName(), err.Error())
			return
		}
	}

	rpc.Accept(listener)
	return
}

// HandleIPFSPath responds to IPFS Hash requests with raw data
func (s *Server) HandleIPFSPath(w http.ResponseWriter, r *http.Request) {
	if s.cfg.API.ReadOnly {
		readOnlyResponse(w, "/ipfs/")
		return
	}

	s.fetchCAFSPath(r.URL.Path, w, r)
}

func (s *Server) fetchCAFSPath(path string, w http.ResponseWriter, r *http.Request) {
	file, err := s.qriNode.Repo.Store().Get(path)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	io.Copy(w, file)
}

// HandleIPNSPath resolves an IPNS entry
func (s *Server) HandleIPNSPath(w http.ResponseWriter, r *http.Request) {
	if s.cfg.API.ReadOnly {
		readOnlyResponse(w, "/ipns/")
		return
	}

	node, err := s.qriNode.IPFSNode()
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("no IPFS node present: %s", err.Error()))
		return
	}

	p, err := node.Namesys.Resolve(r.Context(), r.URL.Path[len("/ipns/"):])
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error resolving IPNS Name: %s", err.Error()))
		return
	}

	file, err := s.qriNode.Repo.Store().Get(p.String())
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	io.Copy(w, file)
}

// helper function
func readOnlyResponse(w http.ResponseWriter, endpoint string) {
	apiutil.WriteErrResponse(w, http.StatusForbidden, fmt.Errorf("qri server is in read-only mode, access to '%s' endpoint is forbidden", endpoint))
}

// HealthCheckHandler is a basic ok response for load balancers & co
// returns the version of qri this node is running, pulled from the lib package
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{ "meta": { "code": 200, "status": "ok", "version":"` + lib.VersionNumber + `" }, "data": [] }`))
}

// NewServerRoutes returns a Muxer that has all API routes
func NewServerRoutes(s *Server) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("/status", s.middleware(HealthCheckHandler))
	m.Handle("/ipfs/", s.middleware(s.HandleIPFSPath))
	m.Handle("/ipns/", s.middleware(s.HandleIPNSPath))

	proh := NewProfileHandlers(s.qriNode, s.cfg.API.ReadOnly)
	m.Handle("/me", s.middleware(proh.ProfileHandler))
	m.Handle("/profile", s.middleware(proh.ProfileHandler))
	m.Handle("/profile/photo", s.middleware(proh.ProfilePhotoHandler))
	m.Handle("/profile/poster", s.middleware(proh.PosterHandler))

	ph := NewPeerHandlers(s.qriNode, s.cfg.API.ReadOnly)
	m.Handle("/peers", s.middleware(ph.PeersHandler))
	m.Handle("/peers/", s.middleware(ph.PeerHandler))

	m.Handle("/connect/", s.middleware(ph.ConnectToPeerHandler))
	m.Handle("/connections", s.middleware(ph.ConnectionsHandler))

	dsh := NewDatasetHandlers(s.qriNode, s.cfg.API.ReadOnly)

	if s.cfg.API.RemoteMode {
		log.Info("This server is running in `remote` mode")
		receivers, err := makeDagReceiver(s.qriNode)
		if err != nil {
			panic(err)
		}

		remh := NewRemoteHandlers(s.qriNode, receivers)
		m.Handle("/dsync/push", s.middleware(remh.ReceiveHandler))
		m.Handle("/dsync", s.middleware(receivers.HTTPHandler()))
		m.Handle("/dsync/complete", s.middleware(remh.CompleteHandler))
	}

	m.Handle("/list", s.middleware(dsh.ListHandler))
	m.Handle("/list/", s.middleware(dsh.PeerListHandler))
	m.Handle("/save", s.middleware(dsh.SaveHandler))
	m.Handle("/save/", s.middleware(dsh.SaveHandler))
	m.Handle("/remove/", s.middleware(dsh.RemoveHandler))
	m.Handle("/me/", s.middleware(dsh.GetHandler))
	m.Handle("/add/", s.middleware(dsh.AddHandler))
	m.Handle("/rename", s.middleware(dsh.RenameHandler))
	m.Handle("/export/", s.middleware(dsh.ZipDatasetHandler))
	m.Handle("/diff", s.middleware(dsh.DiffHandler))
	m.Handle("/body/", s.middleware(dsh.BodyHandler))
	m.Handle("/unpack/", s.middleware(dsh.UnpackHandler))
	m.Handle("/publish/", s.middleware(dsh.PublishHandler))
	m.Handle("/update/", s.middleware(dsh.UpdateHandler))

	renderh := NewRenderHandlers(s.qriNode.Repo)
	m.Handle("/render/", s.middleware(renderh.RenderHandler))

	lh := NewLogHandlers(s.qriNode)
	m.Handle("/history/", s.middleware(lh.LogHandler))

	rgh := NewRegistryHandlers(s.qriNode)
	m.Handle("/registry/datasets", s.middleware(rgh.RegistryDatasetsHandler))
	m.Handle("/registry/", s.middleware(rgh.RegistryHandler))

	sh := NewSearchHandlers(s.qriNode)
	m.Handle("/search", s.middleware(sh.SearchHandler))

	rh := NewRootHandler(dsh, ph)
	m.Handle("/", s.datasetRefMiddleware(s.middleware(rh.Handler)))

	return m
}

// makeDagReceiver constructs a Receivers (HTTP router) from a qri p2p node
func makeDagReceiver(node *p2p.QriNode) (*dsync.Receivers, error) {
	ipfsn, err := node.IPFSNode()
	if err != nil {
		return nil, err
	}
	ng := &dag.NodeGetter{Dag: coreapi.NewCoreAPI(ipfsn).Dag()}

	capi, err := newIPFSCoreAPI(node)
	if err != nil {
		return nil, err
	}
	bapi := capi.Block()

	return dsync.NewReceivers(context.Background(), ng, bapi), nil
}

// newIPFSCoreAPI constructs a CoreAPI (part of ipfs) from a qri p2p node
// copied from qri/actions/dag.go
func newIPFSCoreAPI(node *p2p.QriNode) (capi coreiface.CoreAPI, err error) {
	ipfsn, err := node.IPFSNode()
	if err != nil {
		return nil, err
	}

	return coreapi.NewCoreAPI(ipfsn), nil
}
