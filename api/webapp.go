package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// ServeWebapp launches a webapp server on s.cfg.Webapp.Port
func (s Server) ServeWebapp(ctx context.Context) {
	cfg := s.Config()
	if !cfg.Webapp.Enabled || cfg.Webapp.Port == 0 {
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Webapp.Port))
	if err != nil {
		log.Infof("Webapp listen on port %d error: %s", cfg.Webapp.Port, err)
		return
	}

	m := http.NewServeMux()
	m.Handle("/", s.middleware(s.WebappTemplateHandler))
	m.Handle("/webapp/", s.FrontendHandler("/webapp"))

	webappserver := &http.Server{Handler: m}

	go func() {
		<-ctx.Done()
		log.Info("closing webapp server")
		webappserver.Close()
	}()

	webappserver.Serve(listener)
	return
}

// FrontendHandler resolves the current webapp hash, (fetching the compiled frontend in the process)
// and serves it up as a traditional HTTP endpoint, transparently redirecting requests
// for [prefix]/foo.js to [CAFS Hash]/foo.js
// prefix is the path prefix that should be stripped from the request URL.Path
func (s Server) FrontendHandler(prefix string) http.Handler {
	// TODO - refactor these next two lines. This kicks off a goroutine that checks a IPNS dns entry
	// for the latest hash of the webapp and overwrites with that hash on completion. Problems:
	// * it's mutating its input parameter
	// * it has a race condition with the code below
	// * there's no error handling,
	// * and really the data could be owned by Server and initialized by it,
	//   as there's nothing that necessitates it being within FrontendHandler.
	var webappPath = s.Config().Webapp.EntrypointHash
	go s.resolveWebappPath(&webappPath)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := fmt.Sprintf("%s%s", webappPath, strings.TrimPrefix(r.URL.Path, prefix))
		s.fetchCAFSPath(path, w, r)
	})
}

func (s Server) resolveWebappPath(path *string) {
	node := s.Node()
	cfg := s.Config()
	if cfg.Webapp.EntrypointUpdateAddress == "" {
		log.Debug("no entrypoint update address specified for update checking")
		return
	}

	namesys, err := node.GetIPFSNamesys()
	if err != nil {
		log.Debugf("no IPFS node present to resolve webapp address: %s", err.Error())
		return
	}

	p, err := namesys.Resolve(context.Background(), cfg.Webapp.EntrypointUpdateAddress)
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
func (s Server) WebappTemplateHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(s.Config().Webapp, w, "webapp")
}
