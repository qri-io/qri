package api

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/rpc"

	"github.com/datatogether/api/apiutil"
	"github.com/ipfs/go-datastore"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

var log = golog.Logger("qriapi")

func init() {
	golog.SetLogLevel("qriapi", "info")
}

// Server wraps a qri p2p node, providing traditional access via http
// Create one with New, start it up with Serve
type Server struct {
	// configuration options
	cfg     *config.Config
	qriNode *p2p.QriNode
}

// New creates a new qri server with optional configuration
// calling New() with no options will return the default configuration
// as specified in DefaultConfig
func New(r repo.Repo, options ...func(*config.Config)) (s *Server, err error) {
	cfg := config.Config{}.Default()
	for _, opt := range options {
		opt(cfg)
	}
	// if err := cfg.Validate(); err != nil {
	// 	return nil, fmt.Errorf("server configuration error: %s", err.Error())
	// }

	s = &Server{
		cfg: cfg,
	}

	// allocate a new node
	s.qriNode, err = p2p.NewQriNode(r, func(c *config.P2P) {
		*c = *cfg.P2P
	})
	// func(p2pcfg *config.P2P) {
	// p2pcfg.Online = s.cfg.Online
	// if cfg.BoostrapAddrs != nil {
	// 	p2pcfg.QriBootstrapAddrs = cfg.BoostrapAddrs
	// }
	// })
	if err != nil {
		return s, err
	}

	// bootstrapped := false
	peerBootstrapped := func(peerId string) {
		// if cfg.PostP2POnlineHook != nil && !bootstrapped {
		// go cfg.PostP2POnlineHook(s.qriNode)
		// bootstrapped = true
		// }
	}

	err = s.qriNode.StartOnlineServices(peerBootstrapped)
	if err != nil {
		return nil, fmt.Errorf("error starting P2P service: %s", err.Error())
	}

	return s, nil
}

// Serve starts the server. It will block while the server
// is running
func (s *Server) Serve() (err error) {
	server := &http.Server{}
	server.Handler = NewServerRoutes(s)
	p, err := s.qriNode.Repo.Profile()
	if err != nil {
		return err
	}

	if s.cfg.API.Enabled {
		// log.Info("qri profile id:", s.qriNode.Identity.Pretty())
		info := fmt.Sprintf("connecting to qri:\n  peername: %s\n  QRI ID: %s\n  API port: %s\n  IPFS Addreses:", p.Peername, p.ID, s.cfg.API.Port)
		for _, a := range s.qriNode.EncapsulatedAddresses() {
			info = fmt.Sprintf("%s\n  %s", info, a.String())
		}
		log.Info(info)
	} else {
		log.Info("running qri in offline mode, no peer-2-peer connections")
	}

	go s.ServeRPC()
	go s.ServeWebapp()

	if node, err := s.qriNode.IPFSNode(); err == nil {
		go func() {
			if err := core.CheckVersion(context.Background(), node.Namesys); err == core.ErrUpdateRequired {
				log.Info("This version of qri is out of date, please refer to https://github.com/qri-io/qri/releases/latest for more info")
			} else if err != nil {
				log.Infof("error checking for software update: %s", err.Error())
			}
		}()
	}

	// http.ListenAndServe will not return unless there's an error
	return StartServer(s.cfg.API, server)
}

// ServeRPC checks for a configured RPC port, and registers a listner if so
func (s *Server) ServeRPC() {
	if !s.cfg.RPC.Enabled || s.cfg.RPC.Port == "" {
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", s.cfg.RPC.Port))
	if err != nil {
		log.Infof("RPC listen on port %s error: %s", s.cfg.RPC.Port, err)
		return
	}

	for _, rcvr := range core.Receivers(s.qriNode) {
		if err := rpc.Register(rcvr); err != nil {
			log.Infof("error registering RPC receiver %s: %s", rcvr.CoreRequestsName(), err.Error())
			return
		}
	}

	log.Infof("accepting RPC requests on port %s", s.cfg.RPC.Port)
	rpc.Accept(listener)
	return
}

// ServeWebapp launches a webapp server on s.cfg.Webapp.Port
func (s *Server) ServeWebapp() {
	if !s.cfg.Webapp.Enabled || s.cfg.Webapp.Port == "" {
		return
	}

	go s.resolveWebappPath()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", s.cfg.Webapp.Port))
	if err != nil {
		log.Infof("Webapp listen on port %s error: %s", s.cfg.Webapp.Port, err)
		return
	}

	m := http.NewServeMux()
	m.Handle("/", s.middleware(s.WebappHandler))
	webappserver := &http.Server{Handler: m}

	log.Infof("webapp available on port %s", s.cfg.Webapp.Port)
	webappserver.Serve(listener)
	return
}

func (s *Server) resolveWebappPath() {
	node, err := s.qriNode.IPFSNode()
	if err != nil {
		log.Infof("no IPFS node present to resolve webapp address: %s", err.Error())
		return
	}

	p, err := node.Namesys.Resolve(context.Background(), "/ipns/webapp.qri.io")
	if err != nil {
		log.Infof("error resolving IPNS Name: %s", err.Error())
		return
	}
	log.Debugf("webapp path: %s", p.String())
	s.cfg.Webapp.Scripts = []string{
		fmt.Sprintf("http://localhost:2503%s", p.String()),
	}
}

// HandleIPFSPath responds to IPFS Hash requests with raw data
func (s *Server) HandleIPFSPath(w http.ResponseWriter, r *http.Request) {
	if s.cfg.API.ReadOnly {
		readOnlyResponse(w, "/ipfs/")
		return
	}

	file, err := s.qriNode.Repo.Store().Get(datastore.NewKey(r.URL.Path))
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

	file, err := s.qriNode.Repo.Store().Get(datastore.NewKey(p.String()))
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

// NewServerRoutes returns a Muxer that has all API routes
func NewServerRoutes(s *Server) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("/status", s.middleware(apiutil.HealthCheckHandler))
	m.Handle("/ipfs/", s.middleware(s.HandleIPFSPath))
	m.Handle("/ipns/", s.middleware(s.HandleIPNSPath))

	proh := NewProfileHandlers(s.qriNode.Repo, s.cfg.API.ReadOnly)
	m.Handle("/profile", s.middleware(proh.ProfileHandler))
	m.Handle("/me", s.middleware(proh.ProfileHandler))
	m.Handle("/profile/photo", s.middleware(proh.SetProfilePhotoHandler))
	m.Handle("/profile/poster", s.middleware(proh.SetPosterHandler))

	ph := NewPeerHandlers(s.qriNode.Repo, s.qriNode, s.cfg.API.ReadOnly)
	m.Handle("/peers", s.middleware(ph.PeersHandler))
	m.Handle("/peers/", s.middleware(ph.PeerHandler))

	m.Handle("/connect/", s.middleware(ph.ConnectToPeerHandler))
	m.Handle("/connections", s.middleware(ph.ConnectionsHandler))

	dsh := NewDatasetHandlers(s.qriNode.Repo, s.cfg.API.ReadOnly)

	// TODO - stupid hack for now.
	dsh.DatasetRequests.Node = s.qriNode
	m.Handle("/list", s.middleware(dsh.ListHandler))
	m.Handle("/list/", s.middleware(dsh.PeerListHandler))
	m.Handle("/save", s.middleware(dsh.SaveHandler))
	m.Handle("/save/", s.middleware(dsh.SaveHandler))
	m.Handle("/remove/", s.middleware(dsh.RemoveHandler))
	m.Handle("/me/", s.middleware(dsh.GetHandler))
	m.Handle("/add", s.middleware(dsh.InitHandler))
	m.Handle("/add/", s.middleware(dsh.AddHandler))
	m.Handle("/rename", s.middleware(dsh.RenameHandler))
	m.Handle("/export/", s.middleware(dsh.ZipDatasetHandler))
	m.Handle("/diff", s.middleware(dsh.DiffHandler))
	m.Handle("/data/", s.middleware(dsh.DataHandler))

	hh := NewHistoryHandlers(s.qriNode.Repo)
	// TODO - stupid hack for now.
	hh.HistoryRequests.Node = s.qriNode
	m.Handle("/history/", s.middleware(hh.LogHandler))

	rh := NewRootHandler(dsh, ph)
	m.Handle("/", s.datasetRefMiddleware(s.middleware(rh.Handler)))

	return m
}
