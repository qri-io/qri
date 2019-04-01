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
	"github.com/qri-io/qri/base/fill"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"

	ipld "gx/ipfs/QmR7TcHkR9nxkUorfi8XMTAMLUK7GiP64TWWBzY3aacc1o/go-ipld-format"
	"gx/ipfs/QmUJYo4etAQqFfSS2rarFAE97eNGB8ej64YkRT2SmsYD4r/go-ipfs/core/coreapi"
)

const allowedDagInfoSize uint64 = 10 * 1024 * 1024

// RemoteRequests encapsulates business logic of remote operation
type RemoteRequests struct {
	cli        *rpc.Client
	node       *p2p.QriNode
	Receivers  *dsync.Receivers
	SessionIDs map[string]*dag.Info
}

// NewRemoteRequests creates a RemoteRequests pointer from either a node or an rpc.Client
func NewRemoteRequests(node *p2p.QriNode, cli *rpc.Client) *RemoteRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewRemoteRequests"))
	}
	return &RemoteRequests{
		cli:        cli,
		node:       node,
		SessionIDs: make(map[string]*dag.Info),
	}
}

// CoreRequestsName implements the Requests interface
func (RemoteRequests) CoreRequestsName() string { return "remote" }

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

	dinfo, err := actions.NewDAGInfo(r.node, ref.Path, "")
	if err != nil {
		return err
	}

	location, found := Config.Remotes.Get(p.RemoteName)
	if !found {
		return fmt.Errorf("remote name \"%s\" not found", p.RemoteName)
	}

	data, err := json.Marshal(dinfo)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/dataset", location), bytes.NewReader(data))
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

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var bodyResponse map[string]interface{}
	err = json.Unmarshal(bodyBytes, &bodyResponse)
	if err != nil {
		return err
	}

	dataResponse, ok := bodyResponse["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected: data json")
	}

	var result ReceiveResult
	err = fill.Struct(dataResponse, &result)
	if err != nil {
		return err
	}

	ctx := context.Background()

	ng, err := newNodeGetter(r.node)
	if err != nil {
		return err
	}

	remote := &dsync.HTTPRemote{
		URL: fmt.Sprintf("%s/dsync", location),
	}

	send, err := dsync.NewSend(ctx, ng, dinfo.Manifest, remote)
	if err != nil {
		return err
	}

	err = send.PerformSend(result.SessionID, dinfo.Manifest, result.Diff)
	if err != nil {
		return err
	}

	// TODO(dlong): Pin the data.

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

	dinfo := dag.Info{}
	err = json.Unmarshal([]byte(p.Body), &dinfo)
	if err != nil {
		return err
	}

	// TODO(dlong): Customization for how to decide to accept the dataset.
	if Config.API.RemoteAcceptSizeMax == 0 {
		res.RejectReason = "not accepting any datasets"
		return nil
	}

	// If size is -1, accept any size of dataset. Otherwise, check if the size is allowed.
	if Config.API.RemoteAcceptSizeMax != -1 {
		var totalSize uint64
		for _, s := range dinfo.Sizes {
			totalSize += s
		}

		if totalSize >= uint64(Config.API.RemoteAcceptSizeMax) {
			res.RejectReason = "dataset size too large"
			return nil
		}
	}

	if dinfo.Manifest == nil {
		res.RejectReason = "manifest is nil"
		return nil
	}

	if r.Receivers == nil {
		res.RejectReason = "dag.receivers is nil"
		return nil
	}

	sid, diff, err := r.Receivers.ReqSend(dinfo.Manifest)
	if err != nil {
		res.RejectReason = fmt.Sprintf("could not begin send: %s", err)
		return nil
	}

	// TODO: Timeout for sessionIDs. Add a callback to dsync.Receivers when dsync finishes,
	// then create a version of the dataset for ds_refs.
	r.SessionIDs[sid] = &dinfo
	res.Success = true
	res.SessionID = sid
	res.Diff = diff
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
