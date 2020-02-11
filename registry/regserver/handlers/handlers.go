// Package handlers creates HTTP handler functions for registry interface implementations
package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/qri-io/apiutil"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
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
	}

	node := reg.Remote.Node()
	if node != nil && node.Repo != nil {
		r := node.Repo
		mux.Handle("/registry/feed/home", HomeFeedHandler(r))
		mux.Handle("/registry/dataset/preview/", PreviewHandler("/registry/dataset/preview/", r))
		mux.Handle("/registry/dataset/component/", ComponentHandler(r))
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

// HomeFeedHandler provides access to the home feed
func HomeFeedHandler(r repo.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		refs, err := base.ListDatasets(req.Context(), r, "", feedPageSize, 0, false, true, false)
		if err != nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		res := make([]*dataset.Dataset, len(refs))
		for i, ref := range refs {
			ref.Dataset.Name = ref.Name
			ref.Dataset.Peername = ref.Peername
			res[i] = ref.Dataset
		}

		apiutil.WriteResponse(w, res)
	}
}

// PreviewHandler handles dataset preview requests over HTTP
func PreviewHandler(prefix string, r repo.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ref, err := repo.ParseDatasetRef(strings.TrimPrefix(req.URL.Path, prefix))
		if err != nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		if err := repo.CanonicalizeDatasetRef(r, &ref); err != nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		preview, err := base.CreatePreview(req.Context(), r, reporef.ConvertToDsref(ref))
		if err != nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		apiutil.WriteResponse(w, preview)
	}
}

// ComponentHandler handles dataset component requests over HTTP
func ComponentHandler(r repo.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("unfinished: ComponentHTTPHandler"))
	}
}

// FeedsHTTPHandler gives access to lists of dataset feeds constructed by this
// remote
// TODO (b5) - for now this is just a proxy to list datasets for demonstration
// purposes
// func (r *Remote) FeedsHandler(r repo.Repo) http.HandlerFunc {
// 	return func(w http.ResponseWriter, req *http.Request) {
// 		page := apiutil.PageFromRequest(req)
// 		ctx := req.Context()
// 		refs, err := base.ListDatasets(ctx, r.node.Repo, req.FormValue("term"), page.Limit(), page.Offset(), false, true, false)
// 		if err != nil {
// 			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
// 		}

// 		apiutil.WritePageResponse(w, refs, req, page)
// 	}
// }
