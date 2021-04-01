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
		fmt.Sprintf("%s/%s", p.Endpoint, "{peername}/{name}"),
	}
	if p.Selector {
		routes = append(routes, fmt.Sprintf("%s/%s", p.Endpoint, "{peername}/{name}/{selector}"))
	}
	if !p.ShortRef {
		routes = append(routes, fmt.Sprintf("%s/%s", p.Endpoint, "{peername}/{name}/at/{fs}/{hash}"))
		if p.Selector {
			routes = append(routes, fmt.Sprintf("%s/%s", p.Endpoint, "{peername}/{name}/at/{fs}/{hash}/{selector}"))
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

	m := s.Mux
	if m == nil {
		m = mux.NewRouter()
	}
	m.Use(corsMiddleware(cfg.API.AllowedOrigins))
	m.Use(muxVarsToQueryParamMiddleware)
	m.Use(refStringMiddleware)
	m.Use(token.OAuthTokenMiddleware)

	dsh := NewDatasetHandlers(s.Instance, cfg.API.ReadOnly)

	var routeParams refRouteParams

	// misc endpoints
	m.Handle(lib.AEHome.String(), s.NoLogMiddleware(s.HomeHandler))
	m.Handle(lib.AEHealth.String(), s.NoLogMiddleware(HealthCheckHandler))
	m.Handle(lib.AEIPFS.String(), s.Middleware(s.HandleIPFSPath))
	if !cfg.API.DisableWebui {
		m.Handle(lib.AEWebUI.String(), s.Middleware(WebuiHandler))
	}

	// aggregate endpoints
	m.Handle(lib.AEList.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "collection.list"))).Methods(http.MethodPost)
	m.Handle(lib.AEPeerList.String(), s.Middleware(dsh.PeerListHandler(lib.AEPeerList.String())))
	m.Handle(lib.AESQL.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "sql.exec"))).Methods(http.MethodPost)
	m.Handle(lib.AEDiff.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "diff.diff"))).Methods(http.MethodPost, http.MethodGet)
	m.Handle(lib.AEChanges.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "diff.changes"))).Methods(http.MethodPost, http.MethodGet)

	// access endpoints

	// automation endpoints
	m.Handle(lib.AEApply.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "automation.apply"))).Methods(http.MethodPost)

	// dataset endpoints
	m.Handle(lib.AEComponentStatus.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "dataset.componentstatus"))).Methods(http.MethodPost)
	routeParams = newrefRouteParams(lib.AEGet, false, true, http.MethodGet, http.MethodPost)
	handleRefRoute(m, routeParams, s.Middleware(dsh.GetHandler(lib.AEGet.String())))
	m.Handle(lib.AERename.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "dataset.rename"))).Methods(http.MethodPost, http.MethodPut)
	routeParams = newrefRouteParams(lib.AESave, false, false, http.MethodPost, http.MethodPut)
	handleRefRoute(m, routeParams, s.Middleware(dsh.SaveHandler(lib.AESave.String())))
	routeParams = newrefRouteParams(lib.AEPull, false, false, http.MethodPost, http.MethodPut)
	handleRefRoute(m, routeParams, s.Middleware(dsh.PullHandler(lib.AEPull.NoTrailingSlash())))
	m.Handle(lib.AEPush.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "remote.push"))).Methods(http.MethodPost)
	m.Handle(lib.AERender.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "dataset.render"))).Methods(http.MethodPost)
	m.Handle(lib.AERemoteRemove.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "remote.remove"))).Methods(http.MethodPost)
	routeParams = newrefRouteParams(lib.AERemove, false, false, http.MethodPost, http.MethodDelete)
	handleRefRoute(m, routeParams, s.Middleware(dsh.RemoveHandler(lib.AERemove.String())))
	m.Handle(lib.AEValidate.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "dataset.validate"))).Methods(http.MethodPost)
	m.Handle(lib.AEUnpack.String(), s.Middleware(dsh.UnpackHandler(lib.AEUnpack.NoTrailingSlash())))
	m.Handle(lib.AEManifest.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "dataset.manifest"))).Methods(http.MethodPost)
	m.Handle(lib.AEManifestMissing.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "dataset.manifestmissing"))).Methods(http.MethodPost)
	m.Handle(lib.AEDAGInfo.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "dataset.daginfo"))).Methods(http.MethodPost)

	// peer endpoints
	m.Handle(lib.AEPeers.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "peer.list"))).Methods(http.MethodPost)
	m.Handle(lib.AEPeer.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "peer.info"))).Methods(http.MethodPost)
	m.Handle(lib.AEConnect.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "peer.connect"))).Methods(http.MethodPost)
	m.Handle(lib.AEDisconnect.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "peer.disconnect"))).Methods(http.MethodPost)
	m.Handle(lib.AEConnections.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "peer.connections"))).Methods(http.MethodPost)
	m.Handle(lib.AEConnectedQriProfiles.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "peer.connectedqriprofiles"))).Methods(http.MethodPost)

	// profile endpoints
	m.Handle(lib.AEGetProfile.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "profile.getprofile"))).Methods(http.MethodPost)
	m.Handle(lib.AESetProfile.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "profile.setprofile"))).Methods(http.MethodPost)
	m.Handle(lib.AESetProfilePhoto.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "profile.setprofilePhoto"))).Methods(http.MethodPost)
	m.Handle(lib.AESetPosterPhoto.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "profile.setposterphoto"))).Methods(http.MethodPost)

	// remote endpoints
	m.Handle(lib.AEFeeds.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "remote.feeds"))).Methods(http.MethodPost)
	m.Handle(lib.AEPreview.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "remote.preview"))).Methods(http.MethodPost)
	m.Handle(lib.AERegistryNew.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "registry.createprofile"))).Methods(http.MethodPost)
	m.Handle(lib.AERegistryProve.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "registry.proveprofilekey"))).Methods(http.MethodPost)
	m.Handle(lib.AESearch.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "search.search"))).Methods(http.MethodPost)

	// working directory endpoints
	m.Handle(lib.AEStatus.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.status"))).Methods(http.MethodPost)
	m.Handle(lib.AECanInitDatasetWorkDir.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.caninitdatasetworkdir"))).Methods(http.MethodPost)
	m.Handle(lib.AEInit.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.init"))).Methods(http.MethodPost)
	m.Handle(lib.AECheckout.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.checkout"))).Methods(http.MethodPost)
	m.Handle(lib.AEEnsureRef.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.ensureref"))).Methods(http.MethodPost)
	m.Handle(lib.AERestore.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.restore"))).Methods(http.MethodPost)
	m.Handle(lib.AEFSIWrite.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.write"))).Methods(http.MethodPost)
	m.Handle(lib.AEFSICreateLink.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.createlink"))).Methods(http.MethodPost)
	m.Handle(lib.AEFSIUnlink.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.unlink"))).Methods(http.MethodPost)

	// sync/protocol endpoints
	if cfg.Remote != nil && cfg.Remote.Enabled {
		log.Info("running in `remote` mode")

		m.Handle(lib.AERemoteDSync.String(), s.Middleware(s.Instance.RemoteServer().DsyncHTTPHandler()))
		m.Handle(lib.AERemoteLogSync.String(), s.Middleware(s.Instance.RemoteServer().LogsyncHTTPHandler()))
		m.Handle(lib.AERemoteRefs.String(), s.Middleware(s.Instance.RemoteServer().RefsHTTPHandler()))
	}

	// TODO(aqru): clear up these endpoints up to spec
	m.Handle(lib.AELog.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "log.history"))).Methods(http.MethodPost)
	m.Handle(lib.AEEntries.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "log.log"))).Methods(http.MethodPost)

	return m
}
