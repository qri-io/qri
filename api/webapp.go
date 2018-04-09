package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

// ServeWebapp launches a webapp server on s.cfg.Webapp.Port
func (s *Server) ServeWebapp() {
	if !s.cfg.Webapp.Enabled || s.cfg.Webapp.Port == 0 {
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Webapp.Port))
	if err != nil {
		log.Infof("Webapp listen on port %d error: %s", s.cfg.Webapp.Port, err)
		return
	}

	m := http.NewServeMux()
	m.Handle("/", s.middleware(s.WebappHandler))
	m.Handle("/webapp.js", s.WebappJSHandler())
	webappserver := &http.Server{Handler: m}
	webappserver.Serve(listener)
	return
}

// WebappJSHandler attempts to resolve an update
func (s *Server) WebappJSHandler() http.Handler {
	var webappPath = s.cfg.Webapp.EntrypointHash
	go s.resolveWebappPath(&webappPath)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.fetchIPFSPath(webappPath, w, r)
	})
}

func (s *Server) resolveWebappPath(path *string) {
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
	*path = p.String()
}

// WebappHandler renders the home page
func (s *Server) WebappHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(s.cfg.Webapp, w, "webapp")
}
