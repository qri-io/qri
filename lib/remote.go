package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/rpc"

	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

const allowedDagInfoSize uint64 = 10 * 1024 * 1024

// RemoteRequests encapsulates business logic of remote operation
type RemoteRequests struct {
	cli  *rpc.Client
	node *p2p.QriNode
}

// NewRemoteRequests creates a RemoteRequests pointer from either a node or an rpc.Client
func NewRemoteRequests(node *p2p.QriNode, cli *rpc.Client) *RemoteRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewRemoteRequests"))
	}
	return &RemoteRequests{
		cli:  cli,
		node: node,
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

	// TODO(dlong): Switch to actual dag.Info constructor when it becomes available.
	dinfo, err := newDagInfoTmp(r.node, ref.Path)
	if err != nil {
		return err
	}

	// TODO(dlong): Resolve remote name from p.RemoteName instead of using registry's location.
	location := Config.Registry.Location

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

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	// TODO(dlong): Inspect the remote's response, and then perform dsync.
	fmt.Printf(string(content))

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("error code %d", res.StatusCode)
	}

	*out = true
	return nil
}

// Receive is used to save a dataset when running as a remote. API only, not RPC or command-line.
func (r *RemoteRequests) Receive(p *ReceiveParams, out *bool) (err error) {
	if r.cli != nil {
		return fmt.Errorf("receive cannot be called over RPC")
	}

	dinfo := dagInfoTmp{}
	err = json.Unmarshal([]byte(p.Body), &dinfo)
	if err != nil {
		return err
	}

	fmt.Printf("Received dag.Info:\n")
	fmt.Printf(p.Body)
	fmt.Printf("\n\n")

	var totalSize uint64
	for _, s := range dinfo.Sizes {
		totalSize += s
	}

	// TODO(dlong): Customization for how to decide to accept the dataset.
	if totalSize >= allowedDagInfoSize {
		// TODO(dlong): Instead of merely rejecting, return a message about why.
		*out = false
		return nil
	}

	*out = true
	return nil
}

// TODO(dlong): Switch to actual dag.Info constructor when it becomes available.
func newDagInfoTmp(node *p2p.QriNode, path string) (*dagInfoTmp, error) {
	return &dagInfoTmp{Sizes: []uint64{10, 20, 30}}, nil
}

type dagInfoTmp struct {
	Sizes []uint64
}
