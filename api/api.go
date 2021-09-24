// Package api implements a JSON-API for interacting with a qri node
package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	golog "github.com/ipfs/go-log"
	apiutil "github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/auth/token"
	"github.com/qri-io/qri/lib"
	qhttp "github.com/qri-io/qri/lib/http"
	"github.com/qri-io/qri/lib/websocket"
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
	websocket websocket.Handler
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

	node.LocalStreams.Print(fmt.Sprintf("qri version v%s\nconnecting...\n", APIVersion))

	ws, err := websocket.NewHandler(ctx, s.Instance.Bus(), s.Instance.KeyStore())
	if err != nil {
		return err
	}
	s.websocket = ws
	s.Mux = NewServerRoutes(s)

	p2pConnected := true
	if err := s.Instance.ConnectP2P(ctx); err != nil {
		if !errors.Is(err, lib.ErrP2PDisabled) {
			return err
		}
		p2pConnected = false
	}

	server := &http.Server{
		Handler: s.Mux,
	}

	// TODO(ramfox): check config to see if automation is active
	automationRunning := true
	if err := s.Instance.AutomationListen(ctx); err != nil {
		automationRunning = false
		if !errors.Is(lib.ErrAutomationDisabled, err) {
			return err
		}
	}

	info := "qri is ready.\n"
	if !automationRunning {
		info += "automation is diabled. workflow triggers will not execute\n"
	}
	if !p2pConnected {
		info += "running with no p2p connection\n"
	}
	info += cfg.SummaryString()
	if p2pConnected {
		info += "IPFS Addresses:"
		for _, a := range node.EncapsulatedAddresses() {
			info = fmt.Sprintf("%s\n  %s", info, a.String())
		}
	}
	info += "\n"

	node.LocalStreams.Print(info)

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
		s.websocket.ConnectionHandler(w, r)
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
	Endpoint qhttp.APIEndpoint
	ShortRef bool
	Selector bool
	Methods  []string
}

// newrefRouteParams is a shorthand to generate refRouteParams
func newrefRouteParams(e qhttp.APIEndpoint, sr bool, sel bool, methods ...string) refRouteParams {
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

	m := s.Instance.GiveAPIServer(s.Middleware, []string{})
	m.Use(corsMiddleware(cfg.API.AllowedOrigins))
	m.Use(muxVarsToQueryParamMiddleware)
	m.Use(refStringMiddleware)
	m.Use(token.OAuthTokenMiddleware)

	var routeParams refRouteParams

	// misc endpoints
	m.Handle(AEHome.String(), s.NoLogMiddleware(s.HomeHandler))
	m.Handle(AEHealth.String(), s.NoLogMiddleware(HealthCheckHandler))
	m.Handle(AEIPFS.String(), s.Middleware(s.HandleIPFSPath))
	if cfg.API.Webui {
		m.Handle(AEWebUI.String(), s.Middleware(WebuiHandler))
	}

	// auth endpoints
	m.Handle(AEToken.String(), s.Middleware(TokenHandler(s.Instance))).Methods(http.MethodPost, http.MethodOptions)

	// non POST/json dataset endpoints
	m.Handle(AEGetCSVFullRef.String(), s.Middleware(GetBodyCSVHandler(s.Instance))).Methods(http.MethodGet)
	m.Handle(AEGetCSVShortRef.String(), s.Middleware(GetBodyCSVHandler(s.Instance))).Methods(http.MethodGet)
	routeParams = newrefRouteParams(qhttp.AEGet, false, true, http.MethodGet)
	handleRefRoute(m, routeParams, s.Middleware(GetHandler(s.Instance, qhttp.AEGet.String())))
	m.Handle(AEUnpack.String(), s.Middleware(UnpackHandler(AEUnpack.NoTrailingSlash())))

	// sync/protocol endpoints
	if cfg.RemoteServer != nil && cfg.RemoteServer.Enabled {
		log.Info("running in `remote` mode")

		m.Handle(qhttp.AERemoteDSync.String(), s.Middleware(s.Instance.RemoteServer().DsyncHTTPHandler()))
		m.Handle(qhttp.AERemoteLogSync.String(), s.Middleware(s.Instance.RemoteServer().LogsyncHTTPHandler()))
		m.Handle(qhttp.AERemoteRefs.String(), s.Middleware(s.Instance.RemoteServer().RefsHTTPHandler()))
	}

	return m
}
