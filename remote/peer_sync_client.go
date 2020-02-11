package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	coreiface "github.com/ipfs/interface-go-ipfs-core"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/dag/dsync"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook/logsync"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
	"github.com/qri-io/qri/version"
)

var (
	// ErrNoRemoteClient is returned when no client is allocated
	ErrNoRemoteClient = fmt.Errorf("remote: no client to make remote requests")
	// ErrRemoteNotFound indicates a specified remote couldn't be located
	ErrRemoteNotFound = fmt.Errorf("remote not found")
)

// PeerSyncClient talks to a remote in order to sync peer data
type PeerSyncClient struct {
	pk      crypto.PrivKey
	ds      *dsync.Dsync
	logsync *logsync.Logsync
	capi    coreiface.CoreAPI
	node    *p2p.QriNode
}

// NewClient creates a remote client suitable for syncing peers
func NewClient(node *p2p.QriNode) (c Client, err error) {
	var ds *dsync.Dsync
	capi, capiErr := node.IPFSCoreAPI()
	if capiErr == nil {
		lng, err := dsync.NewLocalNodeGetter(capi)
		if err != nil {
			return nil, err
		}

		ds, err = dsync.New(lng, capi.Block(), func(dsyncConfig *dsync.Config) {
			if host := node.Host(); host != nil {
				dsyncConfig.Libp2pHost = host
			}

			dsyncConfig.PinAPI = capi.Pin()
		})
		if err != nil {
			return nil, err
		}
	} else {
		log.Debug("cannot initialize dsync client, repo isn't using IPFS")
	}

	var ls *logsync.Logsync
	if book := node.Repo.Logbook(); book != nil {
		ls = logsync.New(book, func(logsyncConfig *logsync.Options) {
			if host := node.Host(); host != nil {
				logsyncConfig.Libp2pHost = host
			}
		})
	}

	return &PeerSyncClient{
		pk:      node.Repo.PrivateKey(),
		ds:      ds,
		logsync: ls,
		capi:    capi,
		node:    node,
	}, nil
}

// FetchLogs pulls logbook data from a remote
func (c *PeerSyncClient) FetchLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) (*oplog.Log, error) {
	if c == nil {
		return nil, ErrNoRemoteClient
	}

	if t := addressType(remoteAddr); t == "http" {
		remoteAddr = remoteAddr + "/remote/logsync"
	}
	log.Debugf("fetching logs for %s from %s", ref.Alias(), remoteAddr)
	pull, err := c.logsync.NewPull(ref, remoteAddr)
	if err != nil {
		return nil, err
	}

	return pull.Do(ctx)
}

// CloneLogs pulls logbook data from a remote & stores it locally
func (c *PeerSyncClient) CloneLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	if c == nil {
		return ErrNoRemoteClient
	}

	if t := addressType(remoteAddr); t == "http" {
		remoteAddr = remoteAddr + "/remote/logsync"
	}
	log.Debugf("cloning logs for %s from %s", ref.Alias(), remoteAddr)
	pull, err := c.logsync.NewPull(ref, remoteAddr)
	if err != nil {
		return err
	}

	pull.Merge = true
	_, err = pull.Do(ctx)
	return err
}

// PushLogs pushes logbook data to a remote address
func (c *PeerSyncClient) PushLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	if c == nil {
		return ErrNoRemoteClient
	}

	if t := addressType(remoteAddr); t == "http" {
		remoteAddr = remoteAddr + "/remote/logsync"
	}
	log.Debugf("pushing logs for %s from %s", ref.Alias(), remoteAddr)
	push, err := c.logsync.NewPush(ref, remoteAddr)
	if err != nil {
		return err
	}

	return push.Do(ctx)
}

// RemoveLogs requests a remote remove logbook data from an address
func (c *PeerSyncClient) RemoveLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	if c == nil {
		return ErrNoRemoteClient
	}

	if t := addressType(remoteAddr); t == "http" {
		remoteAddr = remoteAddr + "/remote/logsync"
	}

	log.Debugf("deleting logs for %s from %s", ref.Alias(), remoteAddr)
	return c.logsync.DoRemove(ctx, ref, remoteAddr)
}

// PushDataset pushes the contents of a dataset to a remote
func (c *PeerSyncClient) PushDataset(ctx context.Context, ref reporef.DatasetRef, remoteAddr string) error {
	if c == nil {
		return ErrNoRemoteClient
	}
	if t := addressType(remoteAddr); t == "http" {
		remoteAddr = remoteAddr + "/remote/dsync"
	}
	log.Debugf("pushing dataset %s to %s", ref.Path, remoteAddr)
	push, err := c.ds.NewPush(ref.Path, remoteAddr, true)
	if err != nil {
		return err
	}

	params, err := sigParams(c.pk, ref)
	if err != nil {
		return err
	}
	push.SetMeta(params)

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

	return push.Do(ctx)
}

// PullDataset fetches a dataset from a remote source
func (c *PeerSyncClient) PullDataset(ctx context.Context, ref *reporef.DatasetRef, remoteAddr string) error {
	if c == nil {
		return ErrNoRemoteClient
	}
	log.Debugf("pulling dataset: %s from %s", ref.String(), remoteAddr)

	if ref.Path == "" {
		if err := c.ResolveHeadRef(ctx, ref, remoteAddr); err != nil {
			log.Errorf("resolving head ref: %s", err.Error())
			return err
		}
	}

	params, err := sigParams(c.pk, *ref)
	if err != nil {
		log.Error("generating sig params: ", err)
		return err
	}

	pull, err := c.ds.NewPull(ref.Path, remoteAddr+"/remote/dsync", params)
	if err != nil {
		log.Error("creating pull: ", err)
		return err
	}

	return pull.Do(ctx)
}

// RemoveDataset asks a remote to remove a dataset
func (c *PeerSyncClient) RemoveDataset(ctx context.Context, ref reporef.DatasetRef, remoteAddr string) error {
	if c == nil {
		return ErrNoRemoteClient
	}

	log.Debugf("requesting remove dataset %s from remote %s", ref.Path, remoteAddr)
	params, err := sigParams(c.pk, ref)
	if err != nil {
		return err
	}

	switch addressType(remoteAddr) {
	case "http":
		return removeDatasetHTTP(ctx, params, remoteAddr)
	default:
		return fmt.Errorf("dataset remove requests currently only work over HTTP")
	}
}

// ResolveHeadRef asks a remote to complete a dataset reference, adding the
// latest-known path value
func (c *PeerSyncClient) ResolveHeadRef(ctx context.Context, ref *reporef.DatasetRef, remoteAddr string) error {
	if c == nil {
		return ErrNoRemoteClient
	}

	switch addressType(remoteAddr) {
	case "http":
		return resolveHeadRefHTTP(ctx, ref, remoteAddr)
	default:
		return fmt.Errorf("dataset name resolution currently only works over HTTP")
	}
}

func resolveHeadRefHTTP(ctx context.Context, ref *reporef.DatasetRef, remoteAddr string) error {
	u, err := url.Parse(remoteAddr)
	if err != nil {
		return err
	}

	// TODO (b5) - need to document this convention
	u.Path = "/remote/refs"

	q := u.Query()
	q.Set("peername", ref.Peername)
	q.Set("name", ref.Name)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		errMsg, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("resolving dataset ref from remote failed: %s", string(errMsg))
	}

	return json.NewDecoder(res.Body).Decode(ref)
}

func removeDatasetHTTP(ctx context.Context, params map[string]string, remoteAddr string) error {
	u, err := url.Parse(remoteAddr)
	if err != nil {
		return err
	}

	// TODO (b5) - need to document this convention
	u.Path = "/remote/refs"

	q := u.Query()
	for key, val := range params {
		q.Set(key, val)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		if data, err := ioutil.ReadAll(res.Body); err == nil {
			log.Error("HTTP server remove error response: ", string(data))
		}
		return fmt.Errorf("failed to remove dataset from remote")
	}

	return nil
}

// TODO (b5) - this should return an enumeration
func addressType(remoteAddr string) string {
	// if a valid base58 peerID is passed, we're doing a p2p dsync
	if _, err := peer.IDB58Decode(remoteAddr); err == nil {
		return "p2p"
	} else if strings.HasPrefix(remoteAddr, "http") {
		return "http"
	}

	return ""
}

// ListDatasets shows the reflist of a peer
//
// Deprecated: prefer feed methods instead
func (c *PeerSyncClient) ListDatasets(ctx context.Context, ds *reporef.DatasetRef, term string, offset, limit int) (res []reporef.DatasetRef, err error) {
	if c == nil {
		return nil, ErrNoRemoteClient
	}

	var profiles map[profile.ID]*profile.Profile
	profiles, err = c.node.Repo.Profiles().List()
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error fetching profile: %s", err.Error())
	}

	var pro *profile.Profile
	for _, p := range profiles {
		if ds.ProfileID.String() == p.ID.String() || ds.Peername == p.Peername {
			pro = p
		}
	}
	if err != nil {
		return nil, fmt.Errorf("couldn't find profile: %s", err.Error())
	}
	if pro == nil {
		return nil, fmt.Errorf("profile not found: \"%s\"", ds.Peername)
	}

	if len(pro.PeerIDs) == 0 {
		return nil, fmt.Errorf("couldn't find a peer address for profile: %s", pro.ID)
	}

	res, err = c.node.RequestDatasetsList(ctx, pro.PeerIDs[0], p2p.DatasetsListParams{
		Term:   term,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("error requesting dataset list: %s", err.Error())
	}

	return
}

// AddDataset fetches & pins a dataset to the store, adding it to the list of stored refs
func (c *PeerSyncClient) AddDataset(ctx context.Context, ref *reporef.DatasetRef, remoteAddr string) (err error) {
	if c == nil {
		return ErrNoRemoteClient
	}

	log.Debugf("add dataset %s. remoteAddr: %s", ref.String(), remoteAddr)
	if !ref.Complete() {
		// TODO (b5) - we should remove ResolveHeadRef in favour of a p2p.ResolveDatasetRef
		// head resolution shouldn't require setting up a remote, and should instead be a
		// standard method any qri peer can perform
		if err := c.ResolveHeadRef(ctx, ref, remoteAddr); err != nil {
			return err
		}
	}

	node := c.node

	type addResponse struct {
		Ref   *reporef.DatasetRef
		Error error
	}

	fetchCtx, cancelFetch := context.WithCancel(ctx)
	defer cancelFetch()
	responses := make(chan addResponse)
	tasks := 0

	if remoteAddr != "" {
		tasks++

		refCopy := &reporef.DatasetRef{
			Peername:  ref.Peername,
			ProfileID: ref.ProfileID,
			Name:      ref.Name,
			Path:      ref.Path,
		}

		go func(ref *reporef.DatasetRef) {
			res := addResponse{Ref: ref}

			// always send on responses channel
			defer func() {
				responses <- res
			}()

			if err := c.PullDataset(fetchCtx, ref, remoteAddr); err != nil {
				res.Error = err
				return
			}
			node.LocalStreams.PrintErr("🗼 fetched from registry\n")
			if pinner, ok := node.Repo.Store().(cafs.Pinner); ok {
				err := pinner.Pin(fetchCtx, ref.Path, true)
				res.Error = err
			}
		}(refCopy)
	}

	if node.Online {
		tasks++
		go func() {
			err := base.FetchDataset(fetchCtx, node.Repo, ref, true, true)
			responses <- addResponse{
				Ref:   ref,
				Error: err,
			}
		}()
	}

	if tasks == 0 {
		return fmt.Errorf("no registry configured and node is not online")
	}

	success := false
	for i := 0; i < tasks; i++ {
		res := <-responses
		err = res.Error
		if err == nil {
			cancelFetch()
			success = true
			*ref = *res.Ref
			break
		}
	}

	if !success {
		return fmt.Errorf("add failed: %s", err.Error())
	}

	prevRef, err := node.Repo.GetRef(reporef.DatasetRef{Peername: ref.Peername, Name: ref.Name})
	if err != nil && err == repo.ErrNotFound {
		if err = node.Repo.PutRef(*ref); err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error putting dataset in repo: %s", err.Error())
		}
		return nil
	}
	if err != nil {
		return err
	}

	prevRef.Dataset, err = dsfs.LoadDataset(ctx, node.Repo.Store(), prevRef.Path)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error loading repo dataset: %s", prevRef.Path)
	}

	ref.Dataset, err = dsfs.LoadDataset(ctx, node.Repo.Store(), ref.Path)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error loading added dataset: %s", ref.Path)
	}

	return base.ReplaceRefIfMoreRecent(node.Repo, &prevRef, ref)
}

func (c *PeerSyncClient) signHTTPRequest(req *http.Request) error {
	pk := c.node.Repo.PrivateKey()
	now := fmt.Sprintf("%d", nowFunc().In(time.UTC).Unix())

	// TODO (b5) - we shouldn't be calculating profile IDs here
	peerID, err := calcProfileID(pk)
	if err != nil {
		return err
	}

	b64Sig, err := signString(pk, requestSigningString(now, peerID, req.URL.Path))
	if err != nil {
		return err
	}

	req.Header.Add("timestamp", now)
	req.Header.Add("pid", peerID)
	req.Header.Add("signature", b64Sig)
	req.Header.Add("qri-version", version.String)
	return nil
}

// Feeds fetches the first page of featured & recent feeds in one call
func (c *PeerSyncClient) Feeds(ctx context.Context, remoteAddr string) (map[string][]dsref.VersionInfo, error) {
	if at := addressType(remoteAddr); at != "http" {
		return nil, fmt.Errorf("feeds are only supported over HTTP")
	}

	// TODO (b5) - update registry endpoint name
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/remote/feeds", remoteAddr), nil)
	if err != nil {
		return nil, err
	}

	if err := c.signHTTPRequest(req); err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "no such host") {
			return nil, ErrNoRemoteClient
		}
		return nil, err
	}
	// add response to an envelope
	env := struct {
		Data map[string][]dsref.VersionInfo
		Meta struct {
			Error  string
			Status string
			Code   int
		}
	}{}

	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error %d: %s", res.StatusCode, env.Meta.Error)
	}

	return env.Data, nil
}

// Preview fetches a dataset preview from the registry
func (c *PeerSyncClient) Preview(ctx context.Context, ref dsref.Ref, remoteAddr string) (*dataset.Dataset, error) {
	if at := addressType(remoteAddr); at != "http" {
		return nil, fmt.Errorf("feeds are only supported over HTTP")
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/remote/dataset/preview/%s", remoteAddr, ref.String()), nil)
	if err != nil {
		return nil, err
	}

	if err := c.signHTTPRequest(req); err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "no such host") {
			return nil, ErrRemoteNotFound
		}
		return nil, err
	}
	// add response to an envelope
	env := struct {
		Data *dataset.Dataset
		Meta struct {
			Error  string
			Status string
			Code   int
		}
	}{}

	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error %d: %s", res.StatusCode, env.Meta.Error)
	}

	return env.Data, nil
}
