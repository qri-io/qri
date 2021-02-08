// Package api implements a JSON-API for interacting with a qri node
package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

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
)

func init() {
	golog.SetLogLevel("qriapi", "debug")
}

// Server wraps a qri p2p node, providing traditional access via http
// Create one with New, start it up with Serve
type Server struct {
	*lib.Instance
	Mux       *http.ServeMux
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

// NewServerRoutes returns a Muxer that has all API routes
func NewServerRoutes(s Server) *http.ServeMux {
	cfg := s.Config()

	m := s.Mux
	if m == nil {
		m = http.NewServeMux()
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
	m.Handle(lib.AESave.String(), s.Middleware(dsh.SaveHandler))
	m.Handle(lib.AESaveAlt.String(), s.Middleware(dsh.SaveHandler))
	m.Handle(lib.AERemove.String(), s.Middleware(dsh.RemoveHandler))
	m.Handle(lib.AEGet.String(), s.Middleware(dsh.GetHandler))
	m.Handle(lib.AERename.String(), s.Middleware(dsh.RenameHandler))
	m.Handle(lib.AEDiff.String(), s.Middleware(dsh.DiffHandler))
	m.Handle(lib.AEChanges.String(), s.Middleware(dsh.ChangesHandler))
	// Deprecated, use /get/username/name?component=body or /get/username/name/body.csv
	m.Handle(lib.AEBody.String(), s.Middleware(dsh.BodyHandler))
	m.Handle(lib.AEStats.String(), s.Middleware(dsh.StatsHandler))
	m.Handle(lib.AEUnpack.String(), s.Middleware(dsh.UnpackHandler))

	remClientH := NewRemoteClientHandlers(s.Instance, cfg.API.ReadOnly)
	m.Handle(lib.AEPush.String(), s.Middleware(remClientH.PushHandler))
	m.Handle(lib.AEPull.String(), s.Middleware(dsh.PullHandler))
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

	return m
}
