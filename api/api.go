// Package api implements a JSON-API for interacting with a qri node
package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	golog "github.com/ipfs/go-log"
	apiutil "github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/auth/token"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/version"
)

var (
	log = golog.Logger("qriapi")
	// APIVersion is the version string that is written in API responses
	APIVersion = version.Version
)

const (
	// DefaultTemplateHash is the hash of the default render template
	DefaultTemplateHash = "/ipfs/QmeqeRTf2Cvkqdx4xUdWi1nJB2TgCyxmemsL3H4f1eTBaw"
	// TemplateUpdateAddress is the URI for the template update
	TemplateUpdateAddress = "/ipns/defaulttmpl.qri.io"
)

func init() {
	golog.SetLogLevel("qriapi", "info")
}

// Server wraps a qri p2p node, providing traditional access via http
// Create one with New, start it up with Serve
type Server struct {
	*lib.Instance
	Mux       *mux.Router
	websocket lib.WebsocketHandler
}

// New creates a new qri server from a p2p node & configuration
func New(inst *lib.Instance) Server {
	return Server{
		Instance: inst,
	}
}

// Serve starts the server. It will block while the server is running
func (s Server) Serve(ctx context.Context) (err error) {
	node := s.Node()
	cfg := s.GetConfig()

	ws, err := lib.NewWebsocketHandler(ctx, s.Instance)
	if err != nil {
		return err
	}
	s.websocket = ws
	s.Mux = NewServerRoutes(s)

	if err := s.Instance.Connect(ctx); err != nil {
		return err
	}

	server := &http.Server{
		Handler: s.Mux,
	}

	info := "\n📡  Success! You are now connected to the d.web. Here's your connection details:\n"
	info += cfg.SummaryString()
	info += "IPFS Addresses:"
	for _, a := range node.EncapsulatedAddresses() {
		info = fmt.Sprintf("%s\n  %s", info, a.String())
	}
	info += fmt.Sprintf("\nYou are running Qri v%s", APIVersion)
	info += "\n\n"

	node.LocalStreams.Print(info)

	if cfg.API.DisconnectAfter != 0 {
		log.Infof("disconnecting after %d seconds", cfg.API.DisconnectAfter)
		go func(s *http.Server, t int) {
			<-time.After(time.Second * time.Duration(t))
			log.Infof("disconnecting")
			s.Close()
		}(server, cfg.API.DisconnectAfter)
	}

	go func() {
		<-ctx.Done()
		log.Info("shutting down")
		server.Close()
	}()

	// http.ListenAndServe will not return unless there's an error
	return StartServer(cfg.API, server)
}

// HandleIPFSPath responds to IPFS Hash requests with raw data
func (s *Server) HandleIPFSPath(w http.ResponseWriter, r *http.Request) {
	if s.GetConfig().API.ReadOnly {
		readOnlyResponse(w, "/ipfs/")
		return
	}

	file, err := s.Node().Repo.Filesystem().Get(r.Context(), r.URL.Path)
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

// HomeHandler responds with a health check on the empty path, 404 for
// everything else
func (s *Server) HomeHandler(w http.ResponseWriter, r *http.Request) {
	upgrade := r.Header.Get("Upgrade")
	if upgrade == "websocket" {
		s.websocket.WSConnectionHandler(w, r)
	} else {
		if r.URL.Path == "" || r.URL.Path == "/" {
			HealthCheckHandler(w, r)
			return
		}

		apiutil.NotFoundHandler(w, r)
	}
}

// HealthCheckHandler is a basic ok response for load balancers & co
// returns the version of qri this node is running, pulled from the lib package
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{ "meta": { "code": 200, "status": "ok", "version":"` + APIVersion + `" }, "data": [] }`))
}

// refRouteParams carry a config for a ref based route
type refRouteParams struct {
	Endpoint lib.APIEndpoint
	ShortRef bool
	Selector bool
	Methods  []string
}

// newrefRouteParams is a shorthand to generate refRouteParams
func newrefRouteParams(e lib.APIEndpoint, sr bool, sel bool, methods ...string) refRouteParams {
	return refRouteParams{
		Endpoint: e,
		ShortRef: sr,
		Selector: sel,
		Methods:  methods,
	}
}

func handleRefRoute(m *mux.Router, p refRouteParams, f http.HandlerFunc) {
	routes := []string{
		p.Endpoint.String(),
		fmt.Sprintf("%s/%s", p.Endpoint, "{username}/{name}"),
	}
	if p.Selector {
		routes = append(routes, fmt.Sprintf("%s/%s", p.Endpoint, "{username}/{name}/{selector}"))
	}
	if !p.ShortRef {
		routes = append(routes, fmt.Sprintf("%s/%s", p.Endpoint, "{username}/{name}/at/{fs}/{hash}"))
		if p.Selector {
			routes = append(routes, fmt.Sprintf("%s/%s", p.Endpoint, "{username}/{name}/at/{fs}/{hash}/{selector}"))
		}
	}

	if p.Methods == nil {
		p.Methods = []string{}
	}

	for _, route := range routes {
		if len(p.Methods) > 0 {
			hasOptions := false
			for _, o := range p.Methods {
				if o == http.MethodOptions {
					hasOptions = true
					break
				}
			}
			if !hasOptions {
				p.Methods = append(p.Methods, http.MethodOptions)
			}
			// TODO(b5): this is a band-aid that lets us punt on teaching lib about how to
			// switch on HTTP verbs. I think we should use tricks like this that leverage
			// the gorilla/mux package until we get a better sense of how our API uses
			// HTTP verbs
			m.Handle(route, f).Methods(p.Methods...)
		} else {
			m.Handle(route, f)
		}
	}
}

// NewServerRoutes returns a Muxer that has all API routes
func NewServerRoutes(s Server) *mux.Router {
	cfg := s.GetConfig()

	m := s.Instance.GiveAPIServer(s.Middleware, []string{"dataset.get"})
	m.Use(corsMiddleware(cfg.API.AllowedOrigins))
	m.Use(muxVarsToQueryParamMiddleware)
	m.Use(refStringMiddleware)
	m.Use(token.OAuthTokenMiddleware)

	var routeParams refRouteParams

	// misc endpoints
	m.Handle(lib.AEHome.String(), s.NoLogMiddleware(s.HomeHandler))
	m.Handle(lib.AEHealth.String(), s.NoLogMiddleware(HealthCheckHandler))
	m.Handle(lib.AEIPFS.String(), s.Middleware(s.HandleIPFSPath))
	if !cfg.API.DisableWebui {
		m.Handle(lib.AEWebUI.String(), s.Middleware(WebuiHandler))
	}
	// non POST/json dataset endpoints
	routeParams = newrefRouteParams(lib.AEGet, false, true, http.MethodGet)
	handleRefRoute(m, routeParams, s.Middleware(GetHandler(s.Instance, lib.AEGet.String())))
	m.Handle(lib.AEUnpack.String(), s.Middleware(UnpackHandler(lib.AEUnpack.NoTrailingSlash())))
	// sync/protocol endpoints
	if cfg.Remote != nil && cfg.Remote.Enabled {
		log.Info("running in `remote` mode")

		m.Handle(lib.AERemoteDSync.String(), s.Middleware(s.Instance.RemoteServer().DsyncHTTPHandler()))
		m.Handle(lib.AERemoteLogSync.String(), s.Middleware(s.Instance.RemoteServer().LogsyncHTTPHandler()))
		m.Handle(lib.AERemoteRefs.String(), s.Middleware(s.Instance.RemoteServer().RefsHTTPHandler()))
	}

	return m
}
