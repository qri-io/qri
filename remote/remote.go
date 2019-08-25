// Package remote implements syncronization between qri instances
package remote

import (
	"context"
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
type Hook func(ctx context.Context, ref repo.DatasetRef) error

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
	DatasetPublished Hook
	// called when a client has unpublished a dataset version
	DatasetUnpublished Hook
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
	datasetPublished     Hook
	datasetUnpublished   Hook
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
		datasetPublished:     o.DatasetPublished,
		datasetUnpublished:   o.DatasetUnpublished,
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

		dsyncConfig.RequireAllBlocks = cfg.RequireAllBlocks
		dsyncConfig.PinAPI = capi.Pin()

		dsyncConfig.PreCheck = r.preCheckHook
		dsyncConfig.FinalCheck = r.pushFinalCheckHook
		dsyncConfig.OnComplete = r.onCompleteHook
	})

	return r, err
}

// RemoveDataset handles requests to remove a dataset
func (r *Remote) RemoveDataset(ctx context.Context, params map[string]string) error {
	// TODO (b5):
	return fmt.Errorf("not yet implemented: removing dataset revisions")
}

func (r *Remote) preCheckHook(ctx context.Context, info dag.Info, meta map[string]string) error {
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
		ref, err := r.refFromMeta(meta)
		if err != nil {
			return err
		}
		if err := r.acceptPushPreCheck(ctx, ref); err != nil {
			return err
		}
	}

	return nil
}

func (r *Remote) pushFinalCheckHook(ctx context.Context, info dag.Info, meta map[string]string) error {
	if r.acceptPushFinalCheck != nil {
		ref, err := r.refFromMeta(meta)
		if err != nil {
			return err
		}
		if err := r.acceptPushFinalCheck(ctx, ref); err != nil {
			return err
		}
	}

	return nil
}

func (r *Remote) onCompleteHook(ctx context.Context, info dag.Info, meta map[string]string) error {
	ref, err := r.refFromMeta(meta)
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

	if r.datasetPublished != nil {
		if err = r.datasetPublished(ctx, ref); err != nil {
			return err
		}
	}

	// add completed pushed dataset to our refs
	// TODO (b5) - this could overwrite any FSI links & other ref details,
	// need to investigate
	return r.node.Repo.PutRef(ref)
}

func (r *Remote) refFromMeta(meta map[string]string) (repo.DatasetRef, error) {
	pid, err := profile.IDB58Decode(meta["profileId"])
	if err != nil {
		return repo.DatasetRef{}, err
	}

	ref := repo.DatasetRef{
		Peername:  meta["peername"],
		Name:      meta["name"],
		ProfileID: pid,
		Path:      meta["path"],
	}

	return ref, nil
}

// DsyncHTTPHandler provides an http handler for dsync
func (r *Remote) DsyncHTTPHandler() http.HandlerFunc {
	return dsync.HTTPRemoteHandler(r.dsync)
}
