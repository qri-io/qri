package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/repo"
)

const allowedDagInfoSize uint64 = 10 * 1024 * 1024

// RemoteMethods encapsulates business logic of remote operation
// TODO (b5): switch to using an Instance instead of separate fields
type RemoteMethods struct {
	inst *Instance
}

// NewRemoteMethods creates a RemoteMethods pointer from either a node or an rpc.Client
func NewRemoteMethods(inst *Instance) *RemoteMethods {
	return &RemoteMethods{
		inst: inst,
	}
}

// CoreRequestsName implements the Requests interface
func (*RemoteMethods) CoreRequestsName() string { return "remote" }

// PushToRemote posts a dagInfo to a remote
func (r *RemoteMethods) PushToRemote(p *PushParams, out *bool) error {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.PushToRemote", p, out)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		return err
	}

	dagInfo, err := actions.NewDAGInfo(r.node, ref.Path, "")
	if err != nil {
		return err
	}

	location, found := r.cfg.Remotes.Get(p.RemoteName)
	if !found {
		return fmt.Errorf("remote name \"%s\" not found", p.RemoteName)
	}

	push, err := r.inst.dsync.NewPushManifest(dagInfo.Manifest, location, PushParams.Pin)
	if err != nil {
		return nil
	}

	if err := push.Do(context.Background()); err != nil {
		return nil
	}

	// sessionID, dagDiff, err := actions.DsyncStartPush(r.node, dagInfo, location, &ref)
	// if err != nil {
	// 	return err
	// }

	// err = actions.DsyncSendBlocks(r.node, location, sessionID, dagInfo.Manifest, dagDiff)
	// if err != nil {
	// 	return err
	// }

	// err = actions.DsyncCompletePush(r.node, location, sessionID)
	// if err != nil {
	// 	return err
	// }

	*out = true
	return nil
}

// Receive is used to save a dataset when running as a remote. API only, not RPC or command-line.
func (r *RemoteMethods) Receive(p *ReceiveParams, res *ReceiveResult) (err error) {
	if r.cli != nil {
		return fmt.Errorf("receive cannot be called over RPC")
	}

	res.Success = false

	// TODO(dlong): Customization for how to decide to accept the dataset.
	if r.cfg.API.RemoteAcceptSizeMax == 0 {
		res.RejectReason = "not accepting any datasets"
		return nil
	}

	if p.DagInfo == nil {
		return fmt.Errorf("daginfo is required")
	}

	// If size is -1, accept any size of dataset. Otherwise, check if the size is allowed.
	if r.cfg.API.RemoteAcceptSizeMax != -1 {
		var totalSize uint64
		for _, s := range p.DagInfo.Sizes {
			totalSize += s
		}

		if totalSize >= uint64(r.cfg.API.RemoteAcceptSizeMax) {
			res.RejectReason = "dataset size too large"
			return nil
		}
	}

	if p.DagInfo.Manifest == nil {
		res.RejectReason = "manifest is nil"
		return nil
	}

	if r.Receivers == nil {
		res.RejectReason = "dag.receivers is nil"
		return nil
	}

	sid, diff, err := r.Receivers.ReqSend(p.DagInfo.Manifest)
	if err != nil {
		res.RejectReason = fmt.Sprintf("could not begin send: %s", err)
		return nil
	}

	// Add an entry for this sessionID
	r.lock.Lock()
	r.Sessions[sid] = p
	r.lock.Unlock()

	// Timeout the session
	timeout := r.cfg.API.RemoteAcceptTimeoutMs * time.Millisecond
	if timeout == 0 {
		timeout = time.Second
	}
	go func() {
		time.Sleep(timeout)
		r.lock.Lock()
		defer r.lock.Unlock()
		delete(r.Sessions, sid)
	}()

	// Sucessful response to the client
	res.Success = true
	res.SessionID = sid
	res.Diff = diff
	return nil
}

// Complete is used to complete a dataset that has been pushed to this remote
func (r *RemoteMethods) Complete(p *CompleteParams, res *bool) (err error) {
	sid := p.SessionID
	session, ok := r.Sessions[sid]
	if !ok {
		return fmt.Errorf("session %s not found", sid)
	}

	if session.DagInfo.Manifest == nil || len(session.DagInfo.Manifest.Nodes) == 0 {
		return fmt.Errorf("dataset manifest is invalid")
	}

	path := fmt.Sprintf("/ipfs/%s", session.DagInfo.Manifest.Nodes[0])

	ref := repo.DatasetRef{
		Peername:  session.Peername,
		ProfileID: session.ProfileID,
		Name:      session.Name,
		Path:      path,
		Published: true,
	}

	// Save ref to ds_refs.json
	err = r.node.Repo.PutRef(ref)
	if err != nil {
		return err
	}

	// Pin the dataset in IPFS
	err = base.PinDataset(r.node.Repo, ref)
	if err != nil {
		return err
	}

	r.lock.Lock()
	delete(r.Sessions, sid)
	r.lock.Unlock()

	return nil
}
