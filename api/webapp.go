package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
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
	m.Handle("/", s.middleware(s.WebappTemplateHandler))
	m.Handle("/webapp/", s.FrontendHandler("/webapp"))

	webappserver := &http.Server{Handler: m}
	webappserver.Serve(listener)
	return
}

// FrontendHandler resolves the current webapp hash, (fetching the compiled frontend in the process)
// and serves it up as a traditional HTTP endpoint, transparently redirecting requests
// for [prefix]/foo.js to [CAFS Hash]/foo.js
// prefix is the path prefix that should be stripped from the request URL.Path
func (s *Server) FrontendHandler(prefix string) http.Handler {
	var webappPath = s.cfg.Webapp.EntrypointHash
	go s.resolveWebappPath(&webappPath)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := fmt.Sprintf("%s%s", webappPath, strings.TrimPrefix(r.URL.Path, prefix))
		s.fetchCAFSPath(path, w, r)
	})
}

func (s *Server) resolveWebappPath(path *string) {
	if s.cfg.Webapp.EntrypointUpdateAddress == "" {
		log.Debug("no entrypoint update address specified for update checking")
		return
	}

	node, err := s.qriNode.IPFSNode()
	if err != nil {
		log.Debugf("no IPFS node present to resolve webapp address: %s", err.Error())
		return
	}

	p, err := node.Namesys.Resolve(context.Background(), s.cfg.Webapp.EntrypointUpdateAddress)
	if err != nil {
		log.Infof("error resolving IPNS Name: %s", err.Error())
		return
	}
	if *path != p.String() {
		log.Infof("updating webapp to version: %s", p.String())
	}

	*path = p.String()
}

// WebappTemplateHandler renders the home page
func (s *Server) WebappTemplateHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(s.cfg.Webapp, w, "webapp")
}
