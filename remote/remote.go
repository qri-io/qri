// Package remote implements syncronization between qri instances
package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dag"
	"github.com/qri-io/dag/dsync"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook/logsync"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

var log = golog.Logger("remote")

// Hook is a function called at specific points in the sync cycle
// hook contexts may be populated with request parameters
type Hook func(ctx context.Context, pid profile.ID, ref repo.DatasetRef) error

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
}

// Remote receives requests from other qri nodes to perform actions on their
// behalf
type Remote struct {
	node    *p2p.QriNode
	dsync   *dsync.Dsync
	logsync *logsync.Logsync

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
}

// NewRemote creates a remote
func NewRemote(node *p2p.QriNode, cfg *config.Remote, opts ...func(o *Options)) (*Remote, error) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
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

// ResolveHeadRef fetches the current dataset head path for a given peername and dataset name
func (r *Remote) ResolveHeadRef(ctx context.Context, peername, name string) (*repo.DatasetRef, error) {
	ref := &repo.DatasetRef{
		Peername: peername,
		Name:     name,
	}
	err := repo.CanonicalizeDatasetRef(r.node.Repo, ref)
	return ref, err
}

// RemoveDataset handles requests to remove a dataset
func (r *Remote) RemoveDataset(ctx context.Context, params map[string]string) error {
	pid, ref, err := r.pidAndRefFromMeta(params)
	if err != nil {
		return err
	}
	log.Debug("remove dataset ", ref)

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
	if err := base.RemoveNVersionsFromStore(ctx, r.node.Repo, &ref, -1); err != nil {
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

func (r *Remote) pidAndRefFromMeta(meta map[string]string) (profile.ID, repo.DatasetRef, error) {
	ref := repo.DatasetRef{
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
	return func(ctx context.Context, author logsync.Author, ref dsref.Ref, l *oplog.Log) error {
		if h != nil {
			pid, err := profile.IDB58Decode(author.AuthorID())
			if err != nil {
				return err
			}

			var r repo.DatasetRef
			if ref.String() != "" {
				if r, err = repo.ParseDsref(ref); err != nil {
					return err
				}
			}

			// embed the log oplog pointer in our hook
			ctx = newLogHookContext(ctx, l)

			return h(ctx, pid, r)
		}

		return nil
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

// RefsHTTPHandler handles requests for dataset references
func (r *Remote) RefsHTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			ref := &repo.DatasetRef{
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
