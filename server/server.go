package server

import (
	"fmt"
	"github.com/datatogether/api/apiutil"
	"github.com/qri-io/cafs"
	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/core/datasets"
	"github.com/qri-io/qri/core/peers"
	"github.com/qri-io/qri/core/queries"
	"github.com/qri-io/qri/p2p"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
)

type Server struct {
	cfg *Config
	log *logrus.Logger

	qriNode *p2p.QriNode
	// store   *ipfs.Filestore
}

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

// main app entry point
func (s *Server) Serve() (err error) {
	var store cafs.Filestore
	if s.cfg.LocalIpfs {
		store, err = ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
			cfg.Online = false
			cfg.FsRepoPath = s.cfg.FsStorePath
		})
		if err != nil {
			if strings.Contains(err.Error(), "resource temporarily unavailable") {
				return fmt.Errorf("Couldn't obtain filestore lock. Is an ipfs daemon already running?")
			}
			return err
		}
	}

	qriNode, err := p2p.NewQriNode(store, func(ncfg *p2p.NodeCfg) {
		ncfg.RepoPath = s.cfg.QriRepoPath
		ncfg.Online = s.cfg.Online
	})
	if err != nil {
		return err
	}

	s.qriNode = qriNode

	if s.cfg.Online {
		fmt.Println("qri addresses:")
		for _, a := range s.qriNode.EncapsulatedAddresses() {
			fmt.Printf("  %s\n", a.String())
		}
	} else {
		fmt.Println("running qri in offline mode, no peer-2-peer connections")
	}

	server := &http.Server{}
	server.Handler = s.NewServerRoutes()
	// fire it up!
	s.log.Println("starting server on port", s.cfg.Port)
	// http.ListenAndServe will not return unless there's an error
	return StartServer(s.cfg, server)
}

// NewServerRoutes returns a Muxer that has all API routes.
// This makes for easy testing using httptest, see server_test.go
func (s *Server) NewServerRoutes() *http.ServeMux {
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

	qh := queries.NewHandlers(s.qriNode.Store, s.qriNode.Repo)
	m.Handle("/run", s.middleware(qh.RunHandler))

	return m
}
