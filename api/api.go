// Package api implements a JSON-API for interacting with a qri node
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	golog "github.com/ipfs/go-log"
	apiutil "github.com/qri-io/qri/api/util"
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
	jsonContentType       = "application/json"
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

// NewServerRoutes returns a Muxer that has all API routes
func NewServerRoutes(s Server) *mux.Router {
	cfg := s.Config()

	m := s.Mux
	if m == nil {
		m = mux.NewRouter()
	}

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
	m.Handle(lib.AEDiff.String(), s.Middleware(dsh.DiffHandler))
	m.Handle(lib.AEChanges.String(), s.Middleware(dsh.ChangesHandler))
	m.Handle(lib.AEUnpack.String(), s.Middleware(dsh.UnpackHandler))

	remClientH := NewRemoteClientHandlers(s.Instance, cfg.API.ReadOnly)
	m.Handle(lib.AEPush.String(), s.Middleware(remClientH.PushHandler))
	handleRefRoute(m, lib.AEPull, s.Middleware(dsh.PullHandler))
	m.Handle(lib.AEFeeds.String(), s.Middleware(remClientH.FeedsHandler))
	m.Handle(lib.AEPreview.String(), s.Middleware(remClientH.DatasetPreviewHandler))

	fsih := NewFSIHandlers(s.Instance, cfg.API.ReadOnly)
	m.Handle(lib.AEStatus.String(), s.Middleware(fsih.StatusHandler(lib.AEStatus.NoTrailingSlash())))
	m.Handle(lib.AEWhatChanged.String(), s.Middleware(fsih.WhatChangedHandler(lib.AEWhatChanged.NoTrailingSlash())))
	m.Handle(lib.AEInit.String(), s.Middleware(fsih.InitHandler(lib.AEInit.NoTrailingSlash())))
	m.Handle(lib.AECheckout.String(), s.Middleware(fsih.CheckoutHandler(lib.AECheckout.NoTrailingSlash())))
	m.Handle(lib.AERestore.String(), s.Middleware(fsih.RestoreHandler(lib.AERestore.NoTrailingSlash())))
	m.Handle(lib.AEFSIWrite.String(), s.Middleware(fsih.WriteHandler(lib.AEFSIWrite.NoTrailingSlash())))

	renderh := NewRenderHandlers(s.Instance)
	m.Handle(lib.AERender.String(), s.Middleware(renderh.RenderHandler))
	m.Handle(lib.AERenderAlt.String(), s.Middleware(renderh.RenderHandler))

	lh := NewLogHandlers(s.Instance)
	m.Handle(lib.AEHistory.String(), s.Middleware(lh.LogHandler))

	rch := NewRegistryClientHandlers(s.Instance, cfg.API.ReadOnly)
	m.Handle(lib.AERegistryNew.String(), s.Middleware(rch.CreateProfileHandler))
	m.Handle(lib.AERegistryProve.String(), s.Middleware(rch.ProveProfileKeyHandler))

	sh := NewSearchHandlers(s.Instance)
	m.Handle(lib.AESearch.String(), s.Middleware(sh.SearchHandler))

	sqlh := NewSQLHandlers(s.Instance, cfg.API.ReadOnly)
	m.Handle(lib.AESQL.String(), s.Middleware(sqlh.QueryHandler("/sql")))

	tfh := NewTransformHandlers(s.Instance)
	m.Handle(lib.AEApply.String(), s.Middleware(tfh.ApplyHandler(lib.AEApply.NoTrailingSlash())))

	if !cfg.API.DisableWebui {
		m.Handle(lib.AEWebUI.String(), s.Middleware(WebuiHandler))
	}

	m.Use(refStringMiddleware)

	return m
}

// snoop reads from an io.ReadCloser and restores it so it can be read again
func snoop(body *io.ReadCloser) (io.ReadCloser, error) {
	if body != nil && *body != nil {
		result, err := ioutil.ReadAll(*body)
		(*body).Close()

		if err != nil {
			return nil, err
		}
		if len(result) == 0 {
			return nil, io.EOF
		}

		*body = ioutil.NopCloser(bytes.NewReader(result))
		return ioutil.NopCloser(bytes.NewReader(result)), nil
	}
	return nil, io.EOF
}

var decoder = schema.NewDecoder()

// UnmarshalParams deserialzes a lib req params stuct pointer from an HTTP
// request
func UnmarshalParams(r *http.Request, p interface{}) error {
	// TODO(arqu): once APIs have a strict mapping to Params this line
	// should be removed and should error out on unknown keys
	decoder.IgnoreUnknownKeys(true)
	defer func() {
		if defSetter, ok := p.(lib.NZDefaultSetter); ok {
			defSetter.SetNonZeroDefaults()
		}
	}()

	if r.Method == http.MethodPost || r.Method == http.MethodPut {

		if r.Header.Get("Content-Type") == jsonContentType {
			body, err := snoop(&r.Body)
			if err != nil && err != io.EOF {
				return err
			}
			// this avoids resolving on empty body requests
			// and tries to handle it almost like a GET
			if err != io.EOF {
				if err := json.NewDecoder(body).Decode(p); err != nil {
					return err
				}
			}
		}
	}

	if ru, ok := p.(lib.RequestUnmarshaller); ok {
		return ru.UnmarshalFromRequest(r)
	}

	if err := r.ParseForm(); err != nil {
		return err
	}
	return decoder.Decode(p, r.Form)
}
