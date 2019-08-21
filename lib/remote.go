package lib

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multihash"
	"github.com/qri-io/qri/config"
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
	if r.inst.rpc != nil {
		return r.inst.rpc.Call("DatasetRequests.PushToRemote", p, out)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(r.inst.Repo(), &ref); err != nil {
		return err
	}

	cfg := r.inst.Config()
	dst, err := remoteDsyncDest(cfg, p.RemoteName)
	if err != nil {
		return err
	}

	log.Debugf("publishing %s to %s", ref.Path, dst)
	push, err := r.inst.dsync.NewPush(ref.Path, dst, true)
	if err != nil {
		return err
	}

	params, err := sigParams(r.inst, ref.Path)
	if err != nil {
		return err
	}

	params["peername"] = ref.Peername
	params["name"] = ref.Name
	push.SetMeta(params)

	// TODO (b5) - need contexts yo
	ctx := context.TODO()

	go func() {
		updates := push.Updates()
		for {
			select {
			case update := <-updates:
				fmt.Printf("%d/%d blocks transferred\n", update.CompletedBlocks(), len(update))
				if update.Complete() {
					fmt.Println("done!")
				}
			case <-ctx.Done():
				// don't leak goroutines
				return
			}
		}
	}()

	if err = push.Do(ctx); err != nil {
		return err
	}

	*out = true
	return nil
}

// Receive is used to save a dataset when running as a remote. API only, not RPC or command-line.
func (r *RemoteMethods) Receive(p *ReceiveParams, res *ReceiveResult) (err error) {
	if r.inst.rpc != nil {
		return fmt.Errorf("receive cannot be called over RPC")
	}

	res.Success = false

	// TODO (b5) - restore!
	return fmt.Errorf("need to restore RemoteMethods.Receive")

	// // TODO(dlong): Customization for how to decide to accept the dataset.
	// if r.cfg.API.RemoteAcceptSizeMax == 0 {
	// 	res.RejectReason = "not accepting any datasets"
	// 	return nil
	// }

	// if p.DagInfo == nil {
	// 	return fmt.Errorf("daginfo is required")
	// }

	// // If size is -1, accept any size of dataset. Otherwise, check if the size is allowed.
	// if r.cfg.API.RemoteAcceptSizeMax != -1 {
	// 	var totalSize uint64
	// 	for _, s := range p.DagInfo.Sizes {
	// 		totalSize += s
	// 	}

	// 	if totalSize >= uint64(r.cfg.API.RemoteAcceptSizeMax) {
	// 		res.RejectReason = "dataset size too large"
	// 		return nil
	// 	}
	// }

	// if p.DagInfo.Manifest == nil {
	// 	res.RejectReason = "manifest is nil"
	// 	return nil
	// }

	// if r.Receivers == nil {
	// 	res.RejectReason = "dag.receivers is nil"
	// 	return nil
	// }

	// sid, diff, err := r.Receivers.ReqSend(p.DagInfo.Manifest)
	// if err != nil {
	// 	res.RejectReason = fmt.Sprintf("could not begin send: %s", err)
	// 	return nil
	// }

	// // Add an entry for this sessionID
	// r.lock.Lock()
	// r.Sessions[sid] = p
	// r.lock.Unlock()

	// // Timeout the session
	// timeout := r.cfg.API.RemoteAcceptTimeoutMs * time.Millisecond
	// if timeout == 0 {
	// 	timeout = time.Second
	// }
	// go func() {
	// 	time.Sleep(timeout)
	// 	r.lock.Lock()
	// 	defer r.lock.Unlock()
	// 	delete(r.Sessions, sid)
	// }()

	// // Sucessful response to the client
	// res.Success = true
	// res.SessionID = sid
	// res.Diff = diff
	return nil
}

// Complete is used to complete a dataset that has been pushed to this remote
func (r *RemoteMethods) Complete(p *CompleteParams, res *bool) (err error) {
	// TODO (b5) - restore!
	return fmt.Errorf("TODO (b5) - restore RemoteMethods")
	// sid := p.SessionID
	// session, ok := r.Sessions[sid]
	// if !ok {
	// 	return fmt.Errorf("session %s not found", sid)
	// }

	// if session.DagInfo.Manifest == nil || len(session.DagInfo.Manifest.Nodes) == 0 {
	// 	return fmt.Errorf("dataset manifest is invalid")
	// }

	// path := fmt.Sprintf("/ipfs/%s", session.DagInfo.Manifest.Nodes[0])

	// ref := repo.DatasetRef{
	// 	Peername:  session.Peername,
	// 	ProfileID: session.ProfileID,
	// 	Name:      session.Name,
	// 	Path:      path,
	// 	Published: true,
	// }

	// // Save ref to ds_refs.json
	// err = r.inst.Repo().PutRef(ref)
	// if err != nil {
	// 	return err
	// }

	// // Pin the dataset in IPFS
	// err = base.PinDataset(r.inst.Repo(), ref)
	// if err != nil {
	// 	return err
	// }

	// r.lock.Lock()
	// delete(r.Sessions, sid)
	// r.lock.Unlock()

	return nil
}

func remoteDsyncDest(cfg *config.Config, name string) (dst string, err error) {
	if name == "" {
		if cfg.Registry.Location != "" {
			return cfg.Registry.Location + "/dsync", nil
		}
		return "", fmt.Errorf("no registry specifiied to use as default remote")
	}

	if dst, found := cfg.Remotes.Get(name); found {
		return dst, nil
	}

	return "", fmt.Errorf(`remote name "%s" not found`, name)
}

func sigParams(inst *Instance, cidStr string) (map[string]string, error) {
	pk := inst.Repo().PrivateKey()
	pid, err := calcPeerID(pk)
	if err != nil {
		return nil, err
	}

	now := fmt.Sprintf("%d", time.Now().In(time.UTC).Unix())
	rss := requestSigningString(now, pid, cidStr)

	b64Sig, err := signString(pk, rss)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"peerId":    pid,
		"timestamp": now,
		"cid":       cidStr,
		"signature": b64Sig,
	}, nil
}

func requestSigningString(timestamp, peerID, cidStr string) string {
	return fmt.Sprintf("%s.%s.%s", timestamp, peerID, cidStr)
}

func signString(privKey crypto.PrivKey, str string) (b64Sig string, err error) {
	sigbytes, err := privKey.Sign([]byte(str))
	if err != nil {
		return "", fmt.Errorf("error signing %s", err.Error())
	}

	return base64.StdEncoding.EncodeToString(sigbytes), nil
}

func calcPeerID(privKey crypto.PrivKey) (string, error) {
	pubkeybytes, err := privKey.GetPublic().Bytes()
	if err != nil {
		return "", fmt.Errorf("error getting pubkey bytes: %s", err.Error())
	}

	mh, err := multihash.Sum(pubkeybytes, multihash.SHA2_256, 32)
	if err != nil {
		return "", fmt.Errorf("error summing pubkey: %s", err.Error())
	}

	return mh.B58String(), nil
}
