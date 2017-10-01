package server

import (
	"fmt"
	"github.com/datatogether/api/apiutil"
	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs"
	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/qri/core/datasets"
	"github.com/qri-io/qri/core/peers"
	"github.com/qri-io/qri/core/queries"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
)

// Server wraps a qri p2p node, providing traditional access via http
// Create one with New, start it up with Serve
type Server struct {
	// configuration options
	cfg *Config
	// TODO - remove this logger
	log *logrus.Logger

	qriNode *p2p.QriNode
}

// New creates a new qri server with optional configuration
// calling New() with no options will return the default configuration
// as specified in DefaultConfig
func New(options ...func(*Config)) (*Server, error) {
	cfg := DefaultConfig()
	for _, opt := range options {
		opt(cfg)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("server configuration error: %s", err.Error())
	}

	s := &Server{
		cfg: cfg,
		log: logrus.New(),
	}

	// output to stdout in dev mode
	if s.cfg.Mode == DEVELOP_MODE {
		s.log.Out = os.Stdout
	} else {
		s.log.Out = os.Stderr
	}
	s.log.Level = logrus.InfoLevel
	s.log.Formatter = &logrus.TextFormatter{
		ForceColors: true,
	}

	return s, nil
}

// Serve starts the server. It will block while the server
// is running
func (s *Server) Serve() (err error) {
	var store cafs.Filestore
	var qrepo repo.Repo

	if s.cfg.MemOnly {
		store = memfs.NewMapstore()
		// TODO - refine, adding better identity generation
		// or options for BYO user profile
		qrepo, err = repo.NewMemRepo(
			&profile.Profile{
				Username: "mem user",
			},
			repo.MemPeers{},
			&analytics.Memstore{})
		if err != nil {
			return err
		}
	} else {

		store, err = ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
			// cfg.Online = false
			cfg.Online = s.cfg.Online
			cfg.FsRepoPath = s.cfg.FsStorePath
		})
		if err != nil {
			if strings.Contains(err.Error(), "resource temporarily unavailable") {
				return fmt.Errorf("Couldn't obtain filestore lock. Is an ipfs daemon already running?")
			}

			return err
		}
	}

	// allocate a new node
	qriNode, err := p2p.NewQriNode(store, func(ncfg *p2p.NodeCfg) {
		ncfg.Repo = qrepo
		ncfg.RepoPath = s.cfg.QriRepoPath
		ncfg.Online = s.cfg.Online
	})
	if err != nil {
		return err
	}

	s.qriNode = qriNode

	if s.cfg.Online {
		s.log.Infoln("qri peer id:", s.qriNode.Identity.Pretty())

		s.log.Infoln("qri addresses:")
		for _, a := range s.qriNode.EncapsulatedAddresses() {
			s.log.Infof("  %s", a.String())
			// s.log.Infln(a.Protocols())
		}
	} else {
		s.log.Infoln("running qri in offline mode, no peer-2-peer connections")
	}

	p2p.PrintSwarmAddrs(qriNode)

	server := &http.Server{}
	server.Handler = NewServerRoutes(s)

	s.log.Infoln("starting api server on port", s.cfg.Port)
	// http.ListenAndServe will not return unless there's an error
	return StartServer(s.cfg, server)
}

// NewServerRoutes returns a Muxer that has all API routes
func NewServerRoutes(s *Server) *http.ServeMux {
	m := http.NewServeMux()

	m.HandleFunc("/", WebappHandler)
	m.Handle("/status", s.middleware(apiutil.HealthCheckHandler))

	m.Handle("/ipfs/", s.middleware(s.HandleIPFSPath))

	dsh := datasets.NewHandlers(s.qriNode.Store, s.qriNode.Repo)
	m.Handle("/datasets", s.middleware(dsh.DatasetsHandler))
	m.Handle("/datasets/", s.middleware(dsh.DatasetHandler))
	m.Handle("/data/ipfs/", s.middleware(dsh.StructuredDataHandler))

	ph := peers.NewHandlers(s.qriNode.Repo)
	m.Handle("/peers", s.middleware(ph.PeersHandler))
	m.Handle("/peers/", s.middleware(ph.PeerHandler))

	sh := NewSearchHandlers(s.qriNode.Store, s.qriNode.Repo, s.qriNode)
	m.Handle("/search", s.middleware(sh.SearchHandler))

	qh := queries.NewHandlers(s.qriNode.Store, s.qriNode.Repo)
	m.Handle("/run", s.middleware(qh.RunHandler))

	return m
}
