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
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

var log = golog.Logger("remote")

// Hook is a function
type Hook func(ctx context.Context, pid profile.ID, ref repo.DatasetRef) error

// Options encapsulates runtime configuration for a remote
type Options struct {
	// called when a client requests to push a dataset. The dataset itself
	// will not be accessible at this point, only fields like the name of
	// the dataset, and peer performing the push
	AcceptPushPreCheck Hook
	// called when a dataset has been pushed, but before it's been pinned
	// this is a chance to inspect dataset contents before confirming
	AcceptPushFinalCheck Hook
	// called after successfully publishing a dataset version
	DatasetPushed Hook
	// called when a client has unpublished a dataset version
	DatasetRemoved Hook
	// called when a client pulls a dataset
	DatasetPulled Hook
}

// Remote receives requests from other qri nodes to perform actions on their
// behalf
type Remote struct {
	node  *p2p.QriNode
	dsync *dsync.Dsync

	acceptSizeMax int64
	// TODO (b5) - dsync needs to use timeouts
	acceptTimeoutMs time.Duration

	acceptPushPreCheck   Hook
	acceptPushFinalCheck Hook
	datasetPushed        Hook
	datasetRemoved       Hook
	datasetPulled        Hook
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

		acceptPushPreCheck:   o.AcceptPushPreCheck,
		acceptPushFinalCheck: o.AcceptPushFinalCheck,
		datasetPushed:        o.DatasetPushed,
		datasetRemoved:       o.DatasetRemoved,
		datasetPulled:        o.DatasetPulled,
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

		dsyncConfig.PushPreCheck = r.pushPreCheck
		dsyncConfig.PushFinalCheck = r.pushFinalCheck
		dsyncConfig.PushComplete = r.pushComplete
		dsyncConfig.RemoveCheck = r.removeCheck
		dsyncConfig.GetDagInfoCheck = r.getDagInfo
	})

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

// RemoveDatasetRef handles requests to remove a dataset
func (r *Remote) RemoveDatasetRef(ctx context.Context, params map[string]string) error {
	pid, ref, err := r.pidAndRefFromMeta(params)
	if err != nil {
		return err
	}
	log.Debug("remove dataset ", ref)

	if err := repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		if err == repo.ErrNotFound {
			err = nil
		} else {
			return err
		}
	}

	if r.datasetRemoved != nil {
		if err := r.datasetRemoved(ctx, pid, ref); err != nil {
			return err
		}
	}

	return r.node.Repo.DeleteRef(ref)
}

func (r *Remote) pushPreCheck(ctx context.Context, info dag.Info, meta map[string]string) error {
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

	if r.acceptPushPreCheck != nil {
		pid, ref, err := r.pidAndRefFromMeta(meta)
		if err != nil {
			return err
		}
		if err := r.acceptPushPreCheck(ctx, pid, ref); err != nil {
			return err
		}
	}

	return nil
}

func (r *Remote) pushFinalCheck(ctx context.Context, info dag.Info, meta map[string]string) error {
	if r.acceptPushFinalCheck != nil {
		pid, ref, err := r.pidAndRefFromMeta(meta)
		if err != nil {
			return err
		}
		if err := r.acceptPushFinalCheck(ctx, pid, ref); err != nil {
			return err
		}
	}

	return nil
}

func (r *Remote) pushComplete(ctx context.Context, info dag.Info, meta map[string]string) error {
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

func (r *Remote) removeCheck(ctx context.Context, info dag.Info, meta map[string]string) error {
	pid, ref, err := r.pidAndRefFromMeta(meta)
	if err != nil {
		return err
	}

	if r.datasetRemoved != nil {
		if err = r.datasetRemoved(ctx, pid, ref); err != nil {
			return err
		}
	}
	return nil
}

func (r *Remote) getDagInfo(ctx context.Context, into dag.Info, meta map[string]string) error {
	pid, ref, err := r.pidAndRefFromMeta(meta)
	if err != nil {
		return err
	}

	if r.datasetPulled != nil {
		if err = r.datasetPulled(ctx, pid, ref); err != nil {
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

	if pid, err := profile.IDB58Decode(meta["profileID"]); err == nil {
		ref.ProfileID = pid
	}

	pid, err := profile.IDB58Decode(meta["pid"])

	return pid, ref, err
}

// DsyncHTTPHandler provides an http handler for dsync
func (r *Remote) DsyncHTTPHandler() http.HandlerFunc {
	return dsync.HTTPRemoteHandler(r.dsync)
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
			if err := r.RemoveDatasetRef(req.Context(), params); err != nil {
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
