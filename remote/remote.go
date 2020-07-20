// Package remote implements syncronization between qri instances
package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/apiutil"
	"github.com/qri-io/dag"
	"github.com/qri-io/dag/dsync"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook/logsync"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

var log = golog.Logger("remote")

// Hook is a function called at specific points in the sync cycle
// hook contexts may be populated with request parameters
type Hook func(ctx context.Context, pid profile.ID, ref reporef.DatasetRef) error

// Options encapsulates runtime configuration for a remote
type Options struct {
	// called when a client requests to push a dataset, before any data has been
	// received
	DatasetPushPreCheck Hook
	// called when a dataset has been pushed, but before it's saved
	DatasetPushFinalCheck Hook
	// called after successfully publishing a dataset version
	DatasetPushed Hook
	// called when a client has unpublished a dataset version
	DatasetRemovePreCheck Hook
	// called after a dataset version has been removed
	DatasetRemoved Hook
	// called before a version pull is permitted
	DatasetPullPreCheck Hook
	// called when a client pulls a dataset
	DatasetPulled Hook

	// called before any log data is accepted from a client
	LogPushPreCheck Hook
	// called after a log has been received by a client, before it's saved
	LogPushFinalCheck Hook
	// called after a log has been pushed
	LogPushed Hook
	// called before a log pull is allowed
	LogPullPreCheck Hook
	// called after a log has been pulled
	LogPulled Hook
	// called before a log remove is performed
	LogRemovePreCheck Hook
	// called after a log has been removed
	LogRemoved Hook

	// called before any feed data request is processed
	FeedPreCheck Hook
	// called before a preview request is processed
	PreviewPreCheck Hook

	// Use a custom feeds interface implementation. Default creates a Feeds
	// instance from node.Repo
	Feeds
	// Use a custom previews interface implementation. Default creates a
	// Previews instance from node.Repo
	Previews
}

// Remote receives requests from other qri nodes to perform actions on their
// behalf
type Remote struct {
	node    *p2p.QriNode
	dsync   *dsync.Dsync
	logsync *logsync.Logsync

	Feeds    Feeds
	Previews Previews

	acceptSizeMax int64
	// TODO (b5) - dsync needs to use timeouts
	acceptTimeoutMs time.Duration

	datasetPushPreCheck   Hook
	datasetPushFinalCheck Hook
	datasetPushed         Hook
	datasetRemovePreCheck Hook
	datasetRemoved        Hook
	datasetPullPreCheck   Hook
	datasetPulled         Hook
	FeedPreCheck          Hook
	PreviewPreCheck       Hook
}

// NewRemote creates a remote
func NewRemote(node *p2p.QriNode, cfg *config.Remote, opts ...func(o *Options)) (*Remote, error) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	if node == nil {
		return nil, fmt.Errorf("remote requires a non-nil node")
	}

	r := &Remote{
		node: node,

		acceptSizeMax:   cfg.AcceptSizeMax,
		acceptTimeoutMs: cfg.AcceptTimeoutMs,

		datasetPushPreCheck:   o.DatasetPushPreCheck,
		datasetPushFinalCheck: o.DatasetPushFinalCheck,
		datasetPushed:         o.DatasetPushed,
		datasetRemovePreCheck: o.DatasetRemovePreCheck,
		datasetRemoved:        o.DatasetRemoved,
		datasetPullPreCheck:   o.DatasetPullPreCheck,
		datasetPulled:         o.DatasetPulled,

		FeedPreCheck:    o.FeedPreCheck,
		PreviewPreCheck: o.PreviewPreCheck,
	}

	if o.Feeds != nil {
		r.Feeds = o.Feeds
	} else {
		r.Feeds = RepoFeeds{node.Repo}
	}

	if o.Previews != nil {
		r.Previews = o.Previews
	} else {
		r.Previews = RepoPreviews{node.Repo}
	}

	capi, err := node.IPFSCoreAPI()
	if err != nil {
		return nil, err
	}

	lng, err := dsync.NewLocalNodeGetter(capi)
	if err != nil {
		return nil, err
	}

	r.dsync, err = dsync.New(lng, capi.Block(), func(dsyncConfig *dsync.Config) {
		if host := r.node.Host(); host != nil {
			dsyncConfig.Libp2pHost = host
		}

		dsyncConfig.AllowRemoves = cfg.AllowRemoves
		dsyncConfig.RequireAllBlocks = cfg.RequireAllBlocks
		dsyncConfig.PinAPI = capi.Pin()

		dsyncConfig.PushPreCheck = r.dsPushPreCheck
		dsyncConfig.PushFinalCheck = r.dsPushFinalCheck
		dsyncConfig.PushComplete = r.dsPushComplete
		dsyncConfig.RemoveCheck = r.dsRemovePreCheck
		dsyncConfig.GetDagInfoCheck = r.dsGetDagInfo
	})
	if err != nil {
		return nil, err
	}

	if book := node.Repo.Logbook(); book != nil {
		r.logsync = logsync.New(book, func(lso *logsync.Options) {
			lso.PushPreCheck = r.logHook(o.LogPushPreCheck)
			lso.PushFinalCheck = r.logHook(o.LogPushFinalCheck)
			lso.Pushed = r.logHook(o.LogPushed)
			lso.PullPreCheck = r.logHook(o.LogPullPreCheck)
			lso.Pulled = r.logHook(o.LogPulled)
			lso.RemovePreCheck = r.logHook(o.LogRemovePreCheck)
			lso.Removed = r.logHook(o.LogRemoved)
		})
	}

	return r, err
}

// Node exposes this remote's QriNode
func (r *Remote) Node() *p2p.QriNode {
	if r == nil {
		return nil
	}
	return r.node
}

// Address extracts the address of a remote from a configuration for a given
// remote name
func Address(cfg *config.Config, name string) (addr string, err error) {
	if name == "" {
		if cfg.Registry != nil && cfg.Registry.Location != "" {
			return cfg.Registry.Location, nil
		}
		return "", fmt.Errorf("no registry specifiied to use as default remote")
	}

	if dst, found := cfg.Remotes.Get(name); found {
		return dst, nil
	}

	return "", fmt.Errorf(`remote name "%s" not found`, name)
}

// GoOnline abstracts startDsyncServer, which starts the remote http dsync server
// and adds the dsync protocol to the underlying host
func (r *Remote) GoOnline(ctx context.Context) error {
	return r.startDsyncServer(ctx)
}

// startDsyncServer starts the remote http dsync server and adds the
// dsync protocol to the underlying host
func (r *Remote) startDsyncServer(ctx context.Context) error {
	return r.dsync.StartRemote(ctx)
}

// ResolveHeadRef fetches the current dataset head path for a given peername and dataset name
func (r *Remote) ResolveHeadRef(ctx context.Context, peername, name string) (*reporef.DatasetRef, error) {
	ref := &reporef.DatasetRef{
		Peername: peername,
		Name:     name,
	}

	err := repo.CanonicalizeDatasetRef(r.node.Repo, ref)
	return ref, err
}

// RemoveDataset handles requests to remove a dataset
// currently removes all versions of a dataset
// TODO (ramfox): add `gen` params that indicates how many versions of the dataset, starting
// with the most recent version, we should remove. This should remove the latest version of
// the dataset ref from the refstore and add the (n + 1)th to the refstore
// gen = -1 should indicate that we remove all the dataset versions
func (r *Remote) RemoveDataset(ctx context.Context, params map[string]string) error {
	pid, ref, err := r.pidAndRefFromMeta(params)
	if err != nil {
		return err
	}
	log.Debugf("remove dataset %s", ref)

	// run pre check hook
	if r.datasetRemovePreCheck != nil {
		if err = r.datasetRemovePreCheck(ctx, pid, ref); err != nil {
			return err
		}
	}

	if err := repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		if err == repo.ErrNotFound {
			err = nil
		} else {
			return err
		}
	}

	// remove all the versions of this dataset from the store
	if _, err := base.RemoveNVersionsFromStore(ctx, r.node.Repo, reporef.ConvertToDsref(ref), -1); err != nil {
		return err
	}

	// remove the dataset reference from the repo
	if err := r.node.Repo.DeleteRef(ref); err != nil {
		return err
	}

	// run completed hook
	if r.datasetRemoved != nil {
		if err := r.datasetRemoved(ctx, pid, ref); err != nil {
			return err
		}
	}
	return nil
}

func (r *Remote) dsPushPreCheck(ctx context.Context, info dag.Info, meta map[string]string) error {
	if r.acceptSizeMax == 0 {
		return fmt.Errorf("not accepting any datasets")
	}

	// TODO(dlong): Customization for how to decide to accept the dataset.

	// If size is -1, accept any size of dataset. Otherwise, check if the size is allowed.
	if r.acceptSizeMax != -1 {
		var totalSize uint64
		for _, s := range info.Sizes {
			totalSize += s
		}

		if totalSize >= uint64(r.acceptSizeMax) {
			return fmt.Errorf("dataset size too large")
		}
	}

	if r.datasetPushPreCheck != nil {
		pid, ref, err := r.pidAndRefFromMeta(meta)
		if err != nil {
			return err
		}
		if err := r.datasetPushPreCheck(ctx, pid, ref); err != nil {
			return err
		}
	}

	return nil
}

func (r *Remote) dsPushFinalCheck(ctx context.Context, info dag.Info, meta map[string]string) error {
	if r.datasetPushFinalCheck != nil {
		pid, ref, err := r.pidAndRefFromMeta(meta)
		if err != nil {
			return err
		}
		if err := r.datasetPushFinalCheck(ctx, pid, ref); err != nil {
			return err
		}
	}

	return nil
}

func (r *Remote) dsPushComplete(ctx context.Context, info dag.Info, meta map[string]string) error {
	pid, ref, err := r.pidAndRefFromMeta(meta)
	if err != nil {
		return err
	}

	if err := repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		if err == repo.ErrNotFound {
			err = nil
		} else {
			return err
		}
	}

	if r.datasetPushed != nil {
		if err = r.datasetPushed(ctx, pid, ref); err != nil {
			return err
		}
	}

	// mark ref as published b/c someone just published to us
	ref.Published = true

	// add completed pushed dataset to our refs
	// TODO (b5) - this could overwrite any FSI links & other ref details,
	// need to investigate
	return r.node.Repo.PutRef(ref)
}

func (r *Remote) dsRemovePreCheck(ctx context.Context, info dag.Info, meta map[string]string) error {
	pid, ref, err := r.pidAndRefFromMeta(meta)
	if err != nil {
		return err
	}

	if r.datasetRemovePreCheck != nil {
		if err = r.datasetRemovePreCheck(ctx, pid, ref); err != nil {
			return err
		}
	}
	return nil
}

func (r *Remote) dsGetDagInfo(ctx context.Context, into dag.Info, meta map[string]string) error {
	pid, ref, err := r.pidAndRefFromMeta(meta)
	if err != nil {
		log.Errorf("ref from meta: %s", err.Error())
		return err
	}

	if r.datasetPulled != nil {
		if err = r.datasetPulled(ctx, pid, ref); err != nil {
			log.Errorf("dataset pulled hook: %s", err.Error())
			return err
		}
	}
	return nil
}

func (r *Remote) pidAndRefFromMeta(meta map[string]string) (profile.ID, reporef.DatasetRef, error) {
	ref := reporef.DatasetRef{
		Peername: meta["peername"],
		Name:     meta["name"],
		Path:     meta["path"],
	}

	if pid, decErr := profile.IDB58Decode(meta["profileID"]); decErr == nil {
		ref.ProfileID = pid
	}

	pid, err := profile.IDB58Decode(meta["pid"])

	return pid, ref, err
}

func (r *Remote) logHook(h Hook) logsync.Hook {
	return func(ctx context.Context, author identity.Author, ref dsref.Ref, l *oplog.Log) error {
		if h != nil {
			kid, err := identity.KeyIDFromPub(author.AuthorPubKey())
			if err != nil {
				return err
			}
			pid, err := profile.IDB58Decode(kid)
			if err != nil {
				return err
			}

			var r reporef.DatasetRef
			if ref.String() != "" {
				r = reporef.DatasetRef{
					Peername:  ref.Username,
					ProfileID: profile.IDB58DecodeOrEmpty(ref.ProfileID),
					Name:      ref.Name,
					Path:      ref.Path,
				}
			}

			// embed the log oplog pointer in our hook
			ctx = newLogHookContext(ctx, l)

			return h(ctx, pid, r)
		}

		return nil
	}
}

// AddDefaultRoutes attaches routes a remote client will expect to an HTTP muxer
func (r *Remote) AddDefaultRoutes(mux *http.ServeMux) {
	mux.Handle("/remote/dsync", r.DsyncHTTPHandler())
	mux.Handle("/remote/logsync", r.LogsyncHTTPHandler())
	mux.Handle("/remote/refs", r.RefsHTTPHandler())

	if fs := r.Feeds; fs != nil {
		mux.Handle("/remote/feeds", r.FeedsHTTPHandler())
		mux.Handle("/remote/feeds/", r.FeedHTTPHandler("/remote/feeds/"))
	}
	if ps := r.Previews; ps != nil {
		mux.Handle("/remote/dataset/preview/", r.PreviewHTTPHandler("/remote/dataset/preview/"))
		mux.Handle("/remote/dataset/component/", r.ComponentHTTPHandler("/remote/dataset/component/"))
	}
}

// DsyncHTTPHandler provides an http handler for dsync
func (r *Remote) DsyncHTTPHandler() http.HandlerFunc {
	return dsync.HTTPRemoteHandler(r.dsync)
}

// LogsyncHTTPHandler provides an http handler for synchronizing logs
func (r *Remote) LogsyncHTTPHandler() http.HandlerFunc {
	return logsync.HTTPHandler(r.logsync)
}

// FeedsHTTPHandler provides access to the home feed
func (r *Remote) FeedsHTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		if r.FeedPreCheck != nil {
			id, err := profile.IDB58Decode(req.Header.Get("pid"))
			if err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("missing signature details"))
				return
			}
			if err := r.FeedPreCheck(ctx, id, reporef.DatasetRef{}); err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("missing signature details"))
				return
			}
		}

		feeds, err := r.Feeds.Feeds(ctx, "")
		if err != nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		apiutil.WriteResponse(w, feeds)
	}
}

// max number of items in a page of feed data
const feedPageSize = 30

// FeedHTTPHandler gives access a feed VersionInfos constructed by a remote
func (r *Remote) FeedHTTPHandler(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		if r.FeedPreCheck != nil {
			id, err := profile.IDB58Decode(req.Header.Get("pid"))
			if err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("missing signature details"))
				return
			}
			if err := r.FeedPreCheck(ctx, id, reporef.DatasetRef{}); err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("missing signature details"))
				return
			}
		}

		page := apiutil.PageFromRequest(req)
		refs, err := r.Feeds.Feed(ctx, "", strings.TrimPrefix(req.URL.Path, prefix), page.Offset(), page.Limit())
		if err != nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
		}

		apiutil.WritePageResponse(w, refs, req, page)
	}
}

// PreviewHTTPHandler handles dataset preview requests over HTTP
func (r *Remote) PreviewHTTPHandler(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		if r.PreviewPreCheck != nil {
			id, err := profile.IDB58Decode(req.Header.Get("pid"))
			if err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("missing signature details"))
				return
			}
			if err := r.PreviewPreCheck(ctx, id, reporef.DatasetRef{}); err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("missing signature details"))
				return
			}
		}

		preview, err := r.Previews.Preview(req.Context(), "", strings.TrimPrefix(req.URL.Path, prefix))
		if err != nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		apiutil.WriteResponse(w, preview)
	}
}

// ComponentHTTPHandler handles dataset component requests over HTTP
func (r *Remote) ComponentHTTPHandler(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("unfinished: ComponentHTTPHandler"))
	}
}

// RefsHTTPHandler handles requests for dataset references
func (r *Remote) RefsHTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			ref := &reporef.DatasetRef{
				Peername: req.FormValue("peername"),
				Name:     req.FormValue("name"),
			}
			if err := repo.CanonicalizeDatasetRef(r.node.Repo, ref); err != nil {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(err.Error()))
				return
			}

			res, err := json.Marshal(ref)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(res)
			return
		case "DELETE":
			params := map[string]string{}
			for key := range req.URL.Query() {
				params[key] = req.FormValue(key)
			}
			if err := r.RemoveDataset(req.Context(), params); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			w.WriteHeader(http.StatusOK)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}
