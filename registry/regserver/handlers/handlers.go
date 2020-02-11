// Package handlers creates HTTP handler functions for registry interface implementations
package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/qri-io/apiutil"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/remote"
	"github.com/sirupsen/logrus"
)

var (
	// logger
	log = logrus.New()
)

// SetLogLevel controls how detailed handler logging is
func SetLogLevel(level string) error {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	log.SetLevel(lvl)
	return nil
}

// RouteOptions defines configuration details for NewRoutes
type RouteOptions struct {
	Protector MethodProtector
}

// AddProtector creates a configuration func for passing to NewRoutes
func AddProtector(p MethodProtector) func(o *RouteOptions) {
	return func(o *RouteOptions) {
		o.Protector = p
	}
}

// NewRoutes allocates server handlers along standard routes
func NewRoutes(reg registry.Registry, opts ...func(o *RouteOptions)) *http.ServeMux {
	o := &RouteOptions{
		Protector: NoopProtector(0),
	}
	for _, opt := range opts {
		opt(o)
	}

	pro := o.Protector
	mux := http.NewServeMux()
	mux.HandleFunc("/health", HealthCheckHandler)

	if rem := reg.Remote; rem != nil {
		mux.Handle("/remote/dsync", rem.DsyncHTTPHandler())
		mux.Handle("/remote/logsync", rem.LogsyncHTTPHandler())
		mux.Handle("/remote/refs", rem.RefsHTTPHandler())

		if fs := reg.Remote.Feeds; fs != nil {
			mux.Handle("/remote/feeds", FeedsHandler(fs))
			mux.Handle("/remote/feeds/", FeedHandler("/remote/feeds/", fs))
		}
		if ps := reg.Remote.Previews; ps != nil {
			mux.Handle("/remote/dataset/preview/", PreviewHandler("/remote/dataset/preview/", ps))
			mux.Handle("/remote/dataset/component/", ComponentHandler(ps))
		}
	}

	if ps := reg.Profiles; ps != nil {
		mux.HandleFunc("/registry/profile", logReq(NewProfileHandler(ps)))
		mux.HandleFunc("/registry/profiles", pro.ProtectMethods("POST")(logReq(NewProfilesHandler(ps))))
	}

	if s := reg.Search; s != nil {
		mux.HandleFunc("/registry/search", logReq(NewSearchHandler(s)))
	}

	return mux
}

func logReq(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Infof("%s %s %s", time.Now().Format(time.RFC3339), r.Method, r.URL.Path)
		h.ServeHTTP(w, r)
	}
}

// HealthCheckHandler is a basic "hey I'm fine" for load balancers & co
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"meta":{"code": 200,"status":"ok"},"data":null}`))
}

// max number of items in a page of feed data
const feedPageSize = 30

// FeedsHandler provides access to the home feed
func FeedsHandler(fs remote.Feeds) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		feeds, err := fs.Feeds(req.Context(), "")
		if err != nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		apiutil.WriteResponse(w, feeds)
	}
}

// FeedHandler gives access a feed VersionInfos constructed by a remote
func FeedHandler(prefix string, fs remote.Feeds) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		page := apiutil.PageFromRequest(req)
		refs, err := fs.Feed(req.Context(), "", strings.TrimPrefix(req.URL.Path, prefix), page.Offset(), page.Limit())
		if err != nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
		}

		apiutil.WritePageResponse(w, refs, req, page)
	}
}

// PreviewHandler handles dataset preview requests over HTTP
func PreviewHandler(prefix string, ps remote.Previews) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		preview, err := ps.Preview(req.Context(), "", strings.TrimPrefix(req.URL.Path, prefix))
		if err != nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		apiutil.WriteResponse(w, preview)
	}
}

// ComponentHandler handles dataset component requests over HTTP
func ComponentHandler(fs remote.Previews) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("unfinished: ComponentHTTPHandler"))
	}
}
