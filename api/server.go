package api

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/rpc"

	"github.com/datatogether/api/apiutil"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/api/handlers"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/logging"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// Server wraps a qri p2p node, providing traditional access via http
// Create one with New, start it up with Serve
type Server struct {
	// configuration options
	cfg     *Config
	log     logging.Logger
	qriNode *p2p.QriNode
}

// New creates a new qri server with optional configuration
// calling New() with no options will return the default configuration
// as specified in DefaultConfig
func New(r repo.Repo, options ...func(*Config)) (s *Server, err error) {
	cfg := DefaultConfig()
	for _, opt := range options {
		opt(cfg)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("server configuration error: %s", err.Error())
	}

	s = &Server{
		cfg: cfg,
		log: cfg.Logger,
	}

	// allocate a new node
	s.qriNode, err = p2p.NewQriNode(r, func(ncfg *p2p.NodeCfg) {
		ncfg.Logger = s.log
		ncfg.Online = s.cfg.Online
		if cfg.BoostrapAddrs != nil {
			ncfg.QriBootstrapAddrs = cfg.BoostrapAddrs
		}
	})
	if err != nil {
		return s, err
	}

	bootstrapped := false
	peerBootstrapped := func(peerId string) {
		if cfg.PostP2POnlineHook != nil && !bootstrapped {
			go cfg.PostP2POnlineHook(s.qriNode)
			bootstrapped = true
		}
	}

	err = s.qriNode.StartOnlineServices(peerBootstrapped)
	if err != nil {
		return nil, fmt.Errorf("error starting P2P service: %s", err.Error())
	}
	// p2p.PrintSwarmAddrs(qriNode)
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

	if s.cfg.Online {
		// s.log.Info("qri profile id:", s.qriNode.Identity.Pretty())
		info := fmt.Sprintf("connecting to qri:\n  peername: %s\n  QRI ID: %s\n  API port: %s\n  IPFS Addreses:", p.Peername, p.ID, s.cfg.Port)
		for _, a := range s.qriNode.EncapsulatedAddresses() {
			info = fmt.Sprintf("%s\n  %s", info, a.String())
		}
		s.log.Info(info)
	} else {
		s.log.Info("running qri in offline mode, no peer-2-peer connections")
	}

	go s.ServeRPC()
	// http.ListenAndServe will not return unless there's an error
	return StartServer(s.cfg, server)
}

// ServeRPC checks for a configured RPC port, and registers a listner if so
func (s *Server) ServeRPC() {
	if s.cfg.RPCPort == "" {
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", s.cfg.RPCPort))
	if err != nil {
		s.log.Infof("RPC listen on port %s error: %s", s.cfg.RPCPort, err)
		return
	}

	for _, rcvr := range core.Receivers(s.qriNode) {
		if err := rpc.Register(rcvr); err != nil {
			s.log.Infof("error registering RPC receiver %s: %s", rcvr.CoreRequestsName(), err.Error())
			return
		}
	}

	s.log.Infof("accepting RPC requests on port %s", s.cfg.RPCPort)
	rpc.Accept(listener)
	return
}

// HandleIPFSPath responds to IPFS Hash requests with raw data
func (s *Server) HandleIPFSPath(w http.ResponseWriter, r *http.Request) {
	file, err := s.qriNode.Repo.Store().Get(datastore.NewKey(r.URL.Path))
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	io.Copy(w, file)
}

// NewServerRoutes returns a Muxer that has all API routes
func NewServerRoutes(s *Server) *http.ServeMux {
	m := http.NewServeMux()

	// m.HandleFunc("/", WebappHandler)
	m.Handle("/status", s.middleware(apiutil.HealthCheckHandler))
	m.Handle("/ipfs/", s.middleware(s.HandleIPFSPath))

	proh := handlers.NewProfileHandlers(s.log, s.qriNode.Repo)
	m.Handle("/profile", s.middleware(proh.ProfileHandler))
	m.Handle("/me", s.middleware(proh.ProfileHandler))
	m.Handle("/profile/photo", s.middleware(proh.SetProfilePhotoHandler))
	m.Handle("/profile/poster", s.middleware(proh.SetPosterHandler))

	ph := handlers.NewPeerHandlers(s.log, s.qriNode.Repo, s.qriNode)
	m.Handle("/peers", s.middleware(ph.PeersHandler))
	// TODO: add back connect endpoint
	// m.Handle("/connect/", s.middleware(ph.ConnectToPeerHandler))
	m.Handle("/connections", s.middleware(ph.ConnectionsHandler))

	dsh := handlers.NewDatasetHandlers(s.log, s.qriNode.Repo)
	// TODO - stupid hack for now.
	dsh.DatasetRequests.Node = s.qriNode
	m.Handle("/list", s.middleware(dsh.ListHandler))
	m.Handle("/list/", s.middleware(dsh.PeerListHandler))
	m.Handle("/save/", s.middleware(dsh.SaveHandler))
	m.Handle("/remove/", s.middleware(dsh.RemoveHandler))
	m.Handle("/me/", s.middleware(dsh.GetHandler))
	m.Handle("/add", s.middleware(dsh.AddHandler))
	m.Handle("/rename", s.middleware(dsh.RenameHandler))
	m.Handle("/export/", s.middleware(dsh.ZipDatasetHandler))

	hh := handlers.NewHistoryHandlers(s.log, s.qriNode.Repo)
	m.Handle("/history/", s.middleware(hh.LogHandler))

	rh := handlers.NewRootHandler(dsh, ph)
	m.Handle("/", s.datasetRefMiddleware(s.middleware(rh.Handler)))

	return m
}
