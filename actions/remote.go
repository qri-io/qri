package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/qri-io/dag"
	"github.com/qri-io/dag/dsync"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// DsyncStartPush posts the dataset's dag.Info to the remote to get a sessionID and diff
func DsyncStartPush(node *p2p.QriNode, dagInfo *dag.Info, location string, ref *repo.DatasetRef) (string, *dag.Manifest, error) {
	node.LocalStreams.PrintErr("posting to /dsync/push...\n")
	params := ReceiveParams{
		Peername:  ref.Peername,
		Name:      ref.Name,
		ProfileID: ref.ProfileID,
		DagInfo:   dagInfo,
	}

	data, err := json.Marshal(params)
	if err != nil {
		return "", nil, err
	}

	dsyncPushURL := fmt.Sprintf("%s/dsync/push", location)
	req, err := http.NewRequest("POST", dsyncPushURL, bytes.NewReader(data))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := http.DefaultClient
	res, err := httpClient.Do(req)
	if err != nil {
		return "", nil, err
	}

	if res.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("error code %d: %v", res.StatusCode, rejectionReason(res.Body))
	}

	env := struct{ Data ReceiveResult }{}
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return "", nil, err
	}
	res.Body.Close()

	return env.Data.SessionID, env.Data.Diff, nil
}

// DsyncSendBlocks sends the IPFS blocks to the remote
func DsyncSendBlocks(node *p2p.QriNode, location, sessionID string, manifest, diff *dag.Manifest) error {
	node.LocalStreams.PrintErr("running dsync...\n")
	capi, err := node.IPFSCoreAPI()
	if err != nil {
		return err
	}
	ng := dag.NewNodeGetter(capi.Dag())

	remote := &dsync.HTTPClient{
		URL: fmt.Sprintf("%s/dsync", location),
	}

	ctx := context.Background()
	push, err := dsync.NewPush(ng, &dag.Info{Manifest: manifest}, remote, true)
	if err != nil {
		return err
	}

	if err = push.Do(ctx); err != nil {
		return err
	}

	return nil
}

// DsyncCompletePush completes the send by creating a datasetRef and pinning the dataset
func DsyncCompletePush(node *p2p.QriNode, location, sessionID string) error {
	node.LocalStreams.PrintErr("writing dsref and pinning...\n")
	completeParams := CompleteParams{
		SessionID: sessionID,
	}

	data, err := json.Marshal(completeParams)
	if err != nil {
		return err
	}

	dsyncCompleteURL := fmt.Sprintf("%s/dsync/complete", location)
	req, err := http.NewRequest("POST", dsyncCompleteURL, bytes.NewReader(data))
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

	// Success!
	node.LocalStreams.PrintErr("Success!\n")
	return nil
}

// TODO(dlong): These structs are copied from lib/params.go. We can't use those because
// they are a layer above (importing would cause a cycle), but the http requests here need them.

// ReceiveParams hold parameters for receiving daginfo's when running as a remote
type ReceiveParams struct {
	Peername  string
	Name      string
	ProfileID profile.ID
	DagInfo   *dag.Info
}

// ReceiveResult is the result of receiving a posted dataset when running as a remote
type ReceiveResult struct {
	Success      bool
	RejectReason string
	SessionID    string
	Diff         *dag.Manifest
}

// CompleteParams holds parameters to send when completing a dsync sent to a remote
type CompleteParams struct {
	SessionID string
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
