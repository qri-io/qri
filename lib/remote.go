package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/rpc"

	"github.com/qri-io/dag"
	"github.com/qri-io/dag/dsync"
	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"

	ipld "gx/ipfs/QmR7TcHkR9nxkUorfi8XMTAMLUK7GiP64TWWBzY3aacc1o/go-ipld-format"
	"gx/ipfs/QmUJYo4etAQqFfSS2rarFAE97eNGB8ej64YkRT2SmsYD4r/go-ipfs/core/coreapi"
)

const allowedDagInfoSize uint64 = 10 * 1024 * 1024

// RemoteRequests encapsulates business logic of remote operation
type RemoteRequests struct {
	cli       *rpc.Client
	node      *p2p.QriNode
	Receivers *dsync.Receivers
	Sessions  map[string]*ReceiveParams
}

// NewRemoteRequests creates a RemoteRequests pointer from either a node or an rpc.Client
func NewRemoteRequests(node *p2p.QriNode, cli *rpc.Client) *RemoteRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewRemoteRequests"))
	}
	return &RemoteRequests{
		cli:      cli,
		node:     node,
		Sessions: make(map[string]*ReceiveParams),
	}
}

// CoreRequestsName implements the Requests interface
func (RemoteRequests) CoreRequestsName() string { return "remote" }

// TODO(dlong): Split this function into smaller steps, move them to actions/ or base/ as
// appropriate

// TODO(dlong): Add tests

// PushToRemote posts a dagInfo to a remote
func (r *RemoteRequests) PushToRemote(p *PushParams, out *bool) error {
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

	DagInfo, err := actions.NewDAGInfo(r.node, ref.Path, "")
	if err != nil {
		return err
	}

	location, found := Config.Remotes.Get(p.RemoteName)
	if !found {
		return fmt.Errorf("remote name \"%s\" not found", p.RemoteName)
	}

	// Post the dataset's dag.Info to the remote.
	fmt.Printf("Posting to /dsync/push...\n")

	params := ReceiveParams{
		Peername:  ref.Peername,
		Name:      ref.Name,
		ProfileID: ref.ProfileID,
		DagInfo:   DagInfo,
	}

	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	dsyncPushURL := fmt.Sprintf("%s/dsync/push", location)
	req, err := http.NewRequest("POST", dsyncPushURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := http.DefaultClient
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("error code %d: %v", res.StatusCode, rejectionReason(res.Body))
	}

	env := struct{ Data ReceiveResult }{}
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return err
	}
	res.Body.Close()

	// Run dsync to transfer all of the blocks of the dataset.
	fmt.Printf("Running dsync...\n")

	ng, err := newNodeGetter(r.node)
	if err != nil {
		return err
	}

	remote := &dsync.HTTPRemote{
		URL: fmt.Sprintf("%s/dsync", location),
	}

	ctx := context.Background()
	send, err := dsync.NewSend(ctx, ng, DagInfo.Manifest, remote)
	if err != nil {
		return err
	}

	err = send.PerformSend(env.Data.SessionID, DagInfo.Manifest, env.Data.Diff)
	if err != nil {
		return err
	}

	// Finish the send, pin the dataset in IPFS
	fmt.Printf("Writing dsref and pinning...\n")

	completeParams := CompleteParams{
		SessionID: env.Data.SessionID,
	}

	data, err = json.Marshal(completeParams)
	if err != nil {
		return err
	}

	dsyncCompleteURL := fmt.Sprintf("%s/dsync/complete", location)
	req, err = http.NewRequest("POST", dsyncCompleteURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err = httpClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("error code %d: %v", res.StatusCode, rejectionReason(res.Body))
	}

	// Success!
	fmt.Printf("Success!\n")

	*out = true
	return nil
}

// newNodeGetter generates an ipld.NodeGetter from a QriNode
func newNodeGetter(node *p2p.QriNode) (ng ipld.NodeGetter, err error) {
	ipfsn, err := node.IPFSNode()
	if err != nil {
		return nil, err
	}

	ng = &dag.NodeGetter{Dag: coreapi.NewCoreAPI(ipfsn).Dag()}
	return
}

// Receive is used to save a dataset when running as a remote. API only, not RPC or command-line.
func (r *RemoteRequests) Receive(p *ReceiveParams, res *ReceiveResult) (err error) {
	if r.cli != nil {
		return fmt.Errorf("receive cannot be called over RPC")
	}

	res.Success = false

	// TODO(dlong): Customization for how to decide to accept the dataset.
	if Config.API.RemoteAcceptSizeMax == 0 {
		res.RejectReason = "not accepting any datasets"
		return nil
	}

	if p.DagInfo == nil {
		return fmt.Errorf("daginfo is required")
	}

	// If size is -1, accept any size of dataset. Otherwise, check if the size is allowed.
	if Config.API.RemoteAcceptSizeMax != -1 {
		var totalSize uint64
		for _, s := range p.DagInfo.Sizes {
			totalSize += s
		}

		if totalSize >= uint64(Config.API.RemoteAcceptSizeMax) {
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

	// TODO: Timeout for sessions. Remove sessions when they complete or timeout
	r.Sessions[sid] = p
	res.Success = true
	res.SessionID = sid
	res.Diff = diff
	return nil
}

// Complete is used to complete a dataset that has been pushed to this remote
func (r *RemoteRequests) Complete(p *CompleteParams, res *bool) (err error) {
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

	return nil
}

func rejectionReason(r io.Reader) string {
	text, err := ioutil.ReadAll(r)
	if err != nil {
		return "unknown error"
	}

	var response map[string]interface{}
	err = json.Unmarshal(text, &response)
	if err != nil {
		return fmt.Sprintf("error unmarshalling: %s", string(text))
	}

	meta, ok := response["meta"].(map[string]interface{})
	if !ok {
		return fmt.Sprintf("error unmarshalling: %s", string(text))
	}

	errText, ok := meta["error"].(string)
	if !ok {
		return fmt.Sprintf("error unmarshalling: %s", string(text))
	}

	return errText
}
