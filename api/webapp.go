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

	m := webappMuxer{
		template: s.middleware(s.WebappTemplateHandler),
		webapp:   s.FrontendHandler("/webapp"),
	}

	webappserver := &http.Server{Handler: m}

	go func() {
		<-ctx.Done()
		log.Info("closing webapp server")
		webappserver.Close()
	}()

	webappserver.Serve(listener)
	return
}

type webappMuxer struct {
	webapp, template http.Handler
}

func (h webappMuxer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/webapp") {
		h.webapp.ServeHTTP(w, r)
		return
	}

	h.template.ServeHTTP(w, r)
}

// FrontendHandler fetches the compiled frontend webapp using its hash
// and serves it up as a traditional HTTP endpoint, transparently redirecting requests
// for [prefix]/foo.js to [CAFS Hash]/foo.js
// prefix is the path prefix that should be stripped from the request URL.Path
func (s Server) FrontendHandler(prefix string) http.Handler {
	// TODO -
	// * there's no error handling,
	// * and really the data could be owned by Server and initialized by it,
	//   as there's nothing that necessitates updating the webapp within FrontendHandler.
	if err := s.resolveWebappPath(); err != nil {
		log.Errorf("error resolving webapp path: %s", err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := fmt.Sprintf("%s%s", s.Config().Webapp.EntrypointHash, strings.TrimPrefix(r.URL.Path, prefix))
		log.Info("fetching webapp off the distributed web...")
		s.fetchCAFSPath(path, w, r)
		log.Info("done fetching webapp!")
	})
}

// resolveWebappPath resolved the current webapp hash
func (s Server) resolveWebappPath() error {
	node := s.Node()
	cfg := s.Config()
	if cfg.Webapp.EntrypointUpdateAddress == "" {
		log.Debug("no entrypoint update address specified for update checking")
		return nil
	}

	namesys, err := node.GetIPFSNamesys()
	if err != nil {
		log.Debugf("no IPFS node present to resolve webapp address: %s", err.Error())
		return nil
	}

	p, err := namesys.Resolve(context.Background(), cfg.Webapp.EntrypointUpdateAddress)
	if err != nil {
		return fmt.Errorf("error resolving IPNS Name: %s", err.Error())
	}
	updatedPath := p.String()
	if updatedPath != cfg.Webapp.EntrypointHash {
		log.Infof("updated webapp path to version: %s", updatedPath)
		cfg.Set("webapp.entrypointhash", updatedPath)
		if err := s.ChangeConfig(cfg); err != nil {
			return fmt.Errorf("error updating config: %s", err)
		}
	}
	return nil
}

// WebappTemplateHandler renders the home page
func (s Server) WebappTemplateHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(s.Config().Webapp, w, "webapp")
}
