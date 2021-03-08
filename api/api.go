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
	cfg := s.Config()

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

	go s.ServeRPC(ctx)

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
	if s.Config().API.ReadOnly {
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

func handleRefRoute(m *mux.Router, ae lib.APIEndpoint, f http.HandlerFunc) {
	m.Handle(ae.String(), f)
	m.Handle(fmt.Sprintf("%s/%s", ae, "{peername}/{name}"), f)
	m.Handle(fmt.Sprintf("%s/%s", ae, "{peername}/{name}/{selector}"), f)
	m.Handle(fmt.Sprintf("%s/%s", ae, "{peername}/{name}/at/{fs}/{hash}"), f)
	m.Handle(fmt.Sprintf("%s/%s", ae, "{peername}/{name}/at/{fs}/{hash}/{selector}"), f)
}

// TODO(b5): this is a band-aid that lets us punt on teaching lib about how to
// switch on HTTP verbs. I think we should use tricks like this that leverage
// the gorilla/mux package until we get a better sense of how our API uses
// HTTP verbs
func handleRefRouteMethods(m *mux.Router, ae lib.APIEndpoint, f http.HandlerFunc, methods ...string) {
	m.Handle(ae.String(), f).Methods(methods...)
	m.Handle(fmt.Sprintf("%s/%s", ae, "{peername}/{name}"), f).Methods(methods...)
	m.Handle(fmt.Sprintf("%s/%s", ae, "{peername}/{name}/{selector}"), f).Methods(methods...)
	m.Handle(fmt.Sprintf("%s/%s", ae, "{peername}/{name}/at/{fs}/{hash}"), f).Methods(methods...)
	m.Handle(fmt.Sprintf("%s/%s", ae, "{peername}/{name}/at/{fs}/{hash}/{selector}"), f).Methods(methods...)
}

// NewServerRoutes returns a Muxer that has all API routes
func NewServerRoutes(s Server) *mux.Router {
	cfg := s.Config()

	m := s.Mux
	if m == nil {
		m = mux.NewRouter()
	}
	m.Use(muxVarsToQueryParamMiddleware)
	m.Use(refStringMiddleware)
	m.Use(token.OAuthTokenMiddleware)

	m.Handle(lib.AEHome.String(), s.NoLogMiddleware(s.HomeHandler))
	m.Handle(lib.AEHealth.String(), s.NoLogMiddleware(HealthCheckHandler))
	m.Handle(lib.AEIPFS.String(), s.Middleware(s.HandleIPFSPath))

	proh := NewProfileHandlers(s.Instance, cfg.API.ReadOnly)
	m.Handle(lib.AEMe.String(), s.Middleware(proh.ProfileHandler))
	m.Handle(lib.AEProfile.String(), s.Middleware(proh.ProfileHandler))
	m.Handle(lib.AEProfilePhoto.String(), s.Middleware(proh.ProfilePhotoHandler))
	m.Handle(lib.AEProfilePoster.String(), s.Middleware(proh.PosterHandler))

	ph := NewPeerHandlers(s.Instance, cfg.API.ReadOnly)
	m.Handle(lib.AEPeers.String(), s.Middleware(ph.PeersHandler))
	m.Handle(lib.AEPeer.String(), s.Middleware(ph.PeerHandler))
	m.Handle(lib.AEConnect.String(), s.Middleware(ph.ConnectToPeerHandler))
	m.Handle(lib.AEConnections.String(), s.Middleware(ph.ConnectionsHandler))

	if cfg.Remote != nil && cfg.Remote.Enabled {
		log.Info("running in `remote` mode")

		remh := NewRemoteHandlers(s.Instance)
		m.Handle(lib.AERemoteDSync.String(), s.Middleware(remh.DsyncHandler))
		m.Handle(lib.AERemoteLogSync.String(), s.Middleware(remh.LogsyncHandler))
		m.Handle(lib.AERemoteRefs.String(), s.Middleware(remh.RefsHandler))
	}

	dsh := NewDatasetHandlers(s.Instance, cfg.API.ReadOnly)
	m.Handle(lib.AEList.String(), s.Middleware(dsh.ListHandler))
	m.Handle(lib.AEPeerList.String(), s.Middleware(dsh.PeerListHandler))
	handleRefRoute(m, lib.AESave, s.Middleware(dsh.SaveHandler))
	handleRefRoute(m, lib.AEGet, s.Middleware(dsh.GetHandler))
	handleRefRoute(m, lib.AERemove, s.Middleware(dsh.RemoveHandler))
	m.Handle(lib.AERename.String(), s.Middleware(dsh.RenameHandler))
	handleRefRoute(m, lib.AEValidate, s.Middleware(dsh.ValidateHandler))
	m.Handle(lib.AEDiff.String(), s.Middleware(dsh.DiffHandler))
	m.Handle(lib.AEChanges.String(), s.Middleware(dsh.ChangesHandler(lib.AEChanges.NoTrailingSlash())))
	m.Handle(lib.AEUnpack.String(), s.Middleware(dsh.UnpackHandler))
	m.Handle(lib.AEManifest.String(), s.Middleware(dsh.ManifestHandler))
	m.Handle(lib.AEManifestMissing.String(), s.Middleware(dsh.ManifestMissingHandler))
	m.Handle(lib.AEDAGInfo.String(), s.Middleware(dsh.DAGInfoHandler))

	remClientH := NewRemoteClientHandlers(s.Instance, cfg.API.ReadOnly)
	m.Handle(lib.AEPush.String(), s.Middleware(remClientH.PushHandler))
	handleRefRoute(m, lib.AEPull, s.Middleware(dsh.PullHandler))
	m.Handle(lib.AEFeeds.String(), s.Middleware(remClientH.FeedsHandler))
	m.Handle(lib.AEPreview.String(), s.Middleware(remClientH.DatasetPreviewHandler))

	handleRefRouteMethods(m, lib.AEStatus, s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.status")), http.MethodGet, http.MethodPost)
	handleRefRouteMethods(m, lib.AEWhatChanged, s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.whatchanged")), http.MethodGet, http.MethodPost)
	m.Handle(lib.AEInit.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.init"))).Methods(http.MethodPost)
	m.Handle(fmt.Sprintf("%s/{peername}/{name}", lib.AEInit), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.init"))).Methods(http.MethodPost)
	m.Handle(lib.AECanInitDatasetWorkDir.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.caninitdatasetworkdir")))
	m.Handle(lib.AECheckout.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.checkout"))).Methods(http.MethodPost)
	m.Handle(lib.AERestore.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.restore"))).Methods(http.MethodPost)
	m.Handle(lib.AEFSIWrite.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.write"))).Methods(http.MethodPost)
	m.Handle(lib.AEFSICreateLink.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.createlink"))).Methods(http.MethodPost)
	m.Handle(lib.AEFSIUnlink.String(), s.Middleware(lib.NewHTTPRequestHandler(s.Instance, "fsi.unlink"))).Methods(http.MethodPost)

	renderh := NewRenderHandlers(s.Instance)
	handleRefRoute(m, lib.AERender, s.Middleware(renderh.RenderHandler))

	lh := NewLogHandlers(s.Instance)
	handleRefRoute(m, lib.AEHistory, s.Middleware(lh.LogHandler))
	handleRefRoute(m, lib.AELogbook, s.Middleware(lh.LogbookHandler))
	m.Handle(lib.AELogs.String(), s.Middleware(lh.PlainLogsHandler))
	m.Handle(lib.AELogbookSummary.String(), s.Middleware(lh.LogbookSummaryHandler))

	rch := NewRegistryClientHandlers(s.Instance, cfg.API.ReadOnly)
	m.Handle(lib.AERegistryNew.String(), s.Middleware(rch.CreateProfileHandler))
	m.Handle(lib.AERegistryProve.String(), s.Middleware(rch.ProveProfileKeyHandler))

	sh := NewSearchHandlers(s.Instance)
	m.Handle(lib.AESearch.String(), s.Middleware(sh.SearchHandler))

	sqlh := NewSQLHandlers(s.Instance, cfg.API.ReadOnly)
	m.Handle(lib.AESQL.String(), s.Middleware(sqlh.QueryHandler))

	tfh := NewTransformHandlers(s.Instance)
	m.Handle(lib.AEApply.String(), s.Middleware(tfh.ApplyHandler(lib.AEApply.NoTrailingSlash())))

	if !cfg.API.DisableWebui {
		m.Handle(lib.AEWebUI.String(), s.Middleware(WebuiHandler))
	}

	return m
}
