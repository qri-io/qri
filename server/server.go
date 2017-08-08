package server

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

type Server struct {
	cfg        *Config
	log        *logrus.Logger
	httpServer *http.Server
}

func New(configs ...func(*Config)) *Server {
	cfg := DefaultConfig()
	for _, config := range configs {
		config(cfg)
	}

	s := &Server{
		cfg:        cfg,
		log:        logrus.New(),
		httpServer: &http.Server{},
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

	// connect mux routes to http server
	s.httpServer.Handler = s.NewServerRoutes()
	return s
}

// main app entry point
func (s *Server) Serve() error {
	if err := s.cfg.Validate(); err != nil {
		// panic if the server is missing a vital configuration detail
		return fmt.Errorf("server configuration error: %s", err.Error())
	}

	// fire it up!
	s.log.Println("starting server on port", s.cfg.Port)

	// http.ListenAndServe will not return unless there's an error
	return StartServer(s.cfg, s.httpServer)
}

// NewServerRoutes returns a Muxer that has all API routes.
// This makes for easy testing using httptest, see server_test.go
func (s *Server) NewServerRoutes() *http.ServeMux {
	m := http.NewServeMux()

	m.HandleFunc("/", NotFoundHandler)
	m.Handle("/status", s.middleware(HealthCheckHandler))

	// m.Handle("/users", middleware(UsersHandler))
	// m.Handle("/users/", middleware(UserHandler))
	// m.Handle("/primers", middleware(PrimersHandler))
	// m.Handle("/primers/", middleware(PrimerHandler))
	// m.Handle("/sources", middleware(SourcesHandler))
	// m.Handle("/sources/", middleware(SourceHandler))
	// m.Handle("/urls", middleware(UrlsHandler))
	// m.Handle("/urls/", middleware(UrlHandler))
	// // m.Handle("/links", middleware(LinksHandler))
	// // m.Handle("/links/", middleware(LinkHandler))
	// // m.Handle("/snapshots", middleware(SnapshotsHandler))
	// m.Handle("/coverage", middleware(CoverageHandler))
	// m.Handle("/repositories", middleware(RepositoriesHandler))
	// m.Handle("/repositories/", middleware(RepositoryHandler))
	// // m.Handle("/content", middleware())
	// // m.Handle("/content/", middleware())
	// // m.Handle("/metadata", middleware())
	// // m.Handle("/metadata/", middleware())
	// m.Handle("/tasks", middleware(TasksHandler))
	// m.Handle("/tasks/", middleware(TasksHandler))
	// m.Handle("/collections", middleware(CollectionsHandler))
	// m.Handle("/collections/", middleware(CollectionHandler))
	// m.Handle("/uncrawlables", middleware(UncrawlablesHandler))
	// m.Handle("/uncrawlables/", middleware(UncrawlableHandler))
	// m.HandleFunc("/.well-known/acme-challenge/", CertbotHandler)

	return m
}
