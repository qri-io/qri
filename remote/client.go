package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/dag/dsync"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
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

// Client connects to remotes to perform synchronization tasks
type Client interface {
	// Feeds gets a named set of dataset feeds from a remote, for example a
	// "recent" feed containing a list of datasets the remote has added ordered
	// newest to oldest
	Feeds(ctx context.Context, remoteAddr string) (map[string][]dsref.VersionInfo, error)
	// Feed fetches a named feed of datasets
	Feed(ctx context.Context, remoteAddr, feedName string, page, pageSize int) ([]dsref.VersionInfo, error)
	// Preview fetches a size-bounded subset of a single dataset version,
	// summarizing the contents of the dataset version
	PreviewDatasetVersion(ctx context.Context, ref dsref.Ref, remoteAddr string) (*dataset.Dataset, error)
	// FetchLogs downloads logbook data on a dataset without storing the results
	// locally
	FetchLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) (*oplog.Log, error)
	// NewRemoteRefResolver creates RefResolver backed by network requests to a
	// single remote
	NewRemoteRefResolver(addr string) dsref.Resolver

	// PushDataset synchronizes a dataset with a remote, synchronizing logbook
	// data  and pulling the dataset version specified by ref.Path
	PushDataset(ctx context.Context, ref dsref.Ref, remoteAddr string) error
	// PullDataset fetches & stores a dataset from a remote, synchronizing logbook
	// data and pulling the dataset version data associated with ref.Path
	PullDataset(ctx context.Context, ref *dsref.Ref, remoteAddr string) (*dataset.Dataset, error)
	// RemoveDataset removes a dataset from a remote entirely, delete logbook data
	// on the remote and requesting the remote drop all stored dataset versions
	RemoveDataset(ctx context.Context, ref dsref.Ref, remoteAddr string) error
	// RemoveDatasetVersion asks a remote to stop storing version data for a
	// dataset
	RemoveDatasetVersion(ctx context.Context, ref dsref.Ref, remoteAddr string) error

	// Done returns a channel that the client will send on when the client is
	// closed
	Done() <-chan struct{}
	// DoneErr gives any error that occurred in the shutdown process
	DoneErr() error
	// Shutdown ends the client process early
	Shutdown() <-chan struct{}
}

// client talks to a remote in order to sync peer data
type client struct {
	profile *profile.Profile
	pk      crypto.PrivKey
	ds      *dsync.Dsync
	logsync *logsync.Logsync
	capi    coreiface.CoreAPI
	node    *p2p.QriNode
	events  event.Publisher

	doneCh   chan struct{}
	doneErr  error
	shutdown context.CancelFunc
}

// NewClient creates a remote client suitable for syncing peers
func NewClient(ctx context.Context, node *p2p.QriNode, pub event.Publisher) (c Client, err error) {
	ctx, cancel := context.WithCancel(ctx)
	var ds *dsync.Dsync
	capi, capiErr := node.IPFSCoreAPI()
	if capiErr == nil {
		lng, err := dsync.NewLocalNodeGetter(capi)
		if err != nil {
			cancel()
			return nil, err
		}

		ds, err = dsync.New(lng, capi.Block(), func(dsyncConfig *dsync.Config) {
			if host := node.Host(); host != nil {
				dsyncConfig.Libp2pHost = host
			}

			dsyncConfig.PinAPI = capi.Pin()
		})
		if err != nil {
			cancel()
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

	pro, err := node.Repo.Profile()
	if err != nil {
		log.Debug("cannot get profile from repo, need username for access control on the remote to function")
	}

	cli := &client{
		pk:      node.Repo.PrivateKey(),
		profile: pro,
		ds:      ds,
		logsync: ls,
		capi:    capi,
		node:    node,
		events:  pub,

		doneCh:   make(chan struct{}),
		shutdown: cancel,
	}

	go func() {
		<-ctx.Done()
		// TODO (b5) - return an error here if client is in the process of pulling anything
		cli.doneErr = ctx.Err()
		close(cli.doneCh)
	}()

	return cli, nil
}

// Shutdown closes the client process and returns a channel that will signal
// when it has completely closed and cleaned up
func (c *client) Shutdown() <-chan struct{} {
	c.shutdown()
	return c.Done()
}

// Feeds fetches the first page of featured & recent feeds in one call
func (c *client) Feeds(ctx context.Context, remoteAddr string) (map[string][]dsref.VersionInfo, error) {
	log.Debugf("client.Feeds remoteAddr=%q", remoteAddr)
	if at := addressType(remoteAddr); at != "http" {
		return nil, fmt.Errorf("feeds are only supported over HTTP")
	}

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

// Feeds fetches the first page of featured & recent feeds in one call
func (c *client) Feed(ctx context.Context, remoteAddr, feedName string, page, pageSize int) ([]dsref.VersionInfo, error) {
	log.Debugf("client.Feed remoteAddr=%q feedName=%q page=%d pageSize=%d", remoteAddr, feedName, page, pageSize)
	if at := addressType(remoteAddr); at != "http" {
		return nil, fmt.Errorf("feeds are only supported over HTTP")
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/remote/feeds/%s?page=%d&pageSize=%d", remoteAddr, feedName, page, pageSize), nil)
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
		Data []dsref.VersionInfo
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

// PreviewDatasetVersion fetches a dataset preview from the registry
func (c *client) PreviewDatasetVersion(ctx context.Context, ref dsref.Ref, remoteAddr string) (*dataset.Dataset, error) {
	log.Debugf("client.PreviewDatasetVersion ref=%q remoteAddr=%q", ref, remoteAddr)
	if c == nil {
		return nil, ErrNoRemoteClient
	}
	if at := addressType(remoteAddr); at != "http" {
		return nil, fmt.Errorf("feeds are only supported over HTTP")
	}

	return c.previewDatasetVersionHTTP(ctx, ref, remoteAddr)
}

func (c *client) previewDatasetVersionHTTP(ctx context.Context, ref dsref.Ref, remoteAddr string) (*dataset.Dataset, error) {
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
		log.Errorf("fetching preview from %q: %s", remoteAddr, err)
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

// NewRemoteRefResolver creates a resolver backed by a remote
func (c *client) NewRemoteRefResolver(remoteAddr string) dsref.Resolver {
	log.Debugf("client.NewRemoteRefResolver remoteAddr=%q", remoteAddr)
	if c == nil {
		return nil
	}
	return &remoteRefResolver{cli: c, remoteAddr: remoteAddr}
}

type remoteRefResolver struct {
	cli        *client
	remoteAddr string
}

// ResolveRef implements the dsref.Resolver interface
func (rr *remoteRefResolver) ResolveRef(ctx context.Context, ref *dsref.Ref) (string, error) {
	log.Debugf("client.ResolveRef ref=%q", ref)
	if rr == nil || rr.cli == nil || rr.remoteAddr == "" {
		return rr.remoteAddr, dsref.ErrRefNotFound
	}

	switch addressType(rr.remoteAddr) {
	case "http":
		err := resolveRefHTTP(ctx, ref, rr.remoteAddr)
		return rr.remoteAddr, err
	default:
		return rr.remoteAddr, fmt.Errorf("dataset name resolution currently only works over HTTP")
	}
}

func resolveRefHTTP(ctx context.Context, ref *dsref.Ref, remoteAddr string) error {
	u, err := url.Parse(remoteAddr)
	if err != nil {
		return err
	}

	// TODO (b5) - need to document this convention
	u.Path = "/remote/refs"

	q := u.Query()
	q.Set("username", ref.Username)
	q.Set("name", ref.Name)
	q.Set("path", ref.Path)
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
		errBytes, _ := ioutil.ReadAll(res.Body)
		errMsg := string(errBytes)
		log.Debugf("resolveRefHTTP status code=%d errMsg=%q", res.StatusCode, errMsg)
		if errMsg == dsref.ErrRefNotFound.Error() {
			return dsref.ErrRefNotFound
		}
		return fmt.Errorf("resolving dataset ref from remote %s failed: %s", remoteAddr, errMsg)
	}

	return json.NewDecoder(res.Body).Decode(ref)
}

// FetchLogs pulls logbook data from a remote without persisting it locally
func (c *client) FetchLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) (*oplog.Log, error) {
	log.Debugf("client.FetchLogs ref=%q remoteAddr=%q", ref, remoteAddr)
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

// PushDataset
func (c *client) PushDataset(ctx context.Context, ref dsref.Ref, addr string) error {
	log.Debugf("client.Pushdataset ref=%q addr=%q", ref, addr)
	if c == nil {
		return ErrNoRemoteClient
	}

	if err := c.pushLogs(ctx, ref, addr); err != nil {
		return err
	}
	if err := c.pushDatasetVersion(ctx, ref, addr); err != nil {
		return err
	}

	return c.events.Publish(ctx, event.ETRemoteClientPushDatasetCompleted, event.RemoteEvent{
		Ref:        ref,
		RemoteAddr: addr,
	})
}

// pushLogs pushes logbook data to a remote address
func (c *client) pushLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	log.Debugf("client.pushLogs ref=%q remoteAddr=%q", ref, remoteAddr)
	if t := addressType(remoteAddr); t == "http" {
		remoteAddr = remoteAddr + "/remote/logsync"
	}
	push, err := c.logsync.NewPush(ref, remoteAddr)
	if err != nil {
		return err
	}

	return push.Do(ctx)
}

// PushDatasetVersion pushes the contents of a dataset to a remote
func (c *client) pushDatasetVersion(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	log.Debugf("client.pushDatasetVersion ref=%q remoteAddr=%q", ref, remoteAddr)
	if t := addressType(remoteAddr); t == "http" {
		remoteAddr = remoteAddr + "/remote/dsync"
	}
	push, err := c.ds.NewPush(ref.Path, remoteAddr, true)
	if err != nil {
		return err
	}

	params, err := sigParams(c.pk, c.profile.Peername, ref)
	if err != nil {
		return err
	}
	push.SetMeta(params)

	go func() {
		updates := push.Updates()
		for {
			select {
			case update := <-updates:
				go func() {
					prog := event.RemoteEvent{
						Ref:        ref,
						RemoteAddr: remoteAddr,
						Progress:   update,
					}
					if err := c.events.Publish(ctx, event.ETRemoteClientPushVersionProgress, prog); err != nil {
						log.Debugf("publishing eventType=%q error=%q", event.ETRemoteClientPushVersionProgress, err)
					}
				}()
			case <-ctx.Done():
				return
			}
		}
	}()

	if err := push.Do(ctx); err != nil {
		return err
	}

	return c.events.Publish(ctx, event.ETRemoteClientPushVersionCompleted, event.RemoteEvent{
		Ref:        ref,
		RemoteAddr: remoteAddr,
	})
}

// PullDataset fetches & pins a dataset to the store, adding it to the list of
// stored refs
// TODO (b5) - remove all p2p calls from here into the p2p subsystem in a future
// refactor
func (c *client) PullDataset(ctx context.Context, ref *dsref.Ref, remoteAddr string) (ds *dataset.Dataset, err error) {
	log.Debugf("client.PullDataset ref=%q addr=%q", ref, remoteAddr)
	if c == nil {
		return nil, ErrNoRemoteClient
	}

	node := c.node

	if err := c.pullLogs(ctx, *ref, remoteAddr); err != nil {
		log.Debugf("client.pullLogs error=%q", err)
		return nil, err
	}

	if err := c.pullDatasetVersion(ctx, ref, remoteAddr); err != nil {
		log.Debugf("client.pullDatasetVersion error=%q", err)
		return nil, err
	}
	node.LocalStreams.PrintErr(fmt.Sprintf("🗼 fetched from remote %q\n", remoteAddr))

	err = c.events.Publish(ctx, event.ETRemoteClientPullDatasetCompleted, event.RemoteEvent{
		Ref:        *ref,
		RemoteAddr: remoteAddr,
	})
	if err != nil {
		return nil, err
	}

	// TODO (b5) - contents of this functino below here be moved into an event
	// handler subscribed to event.ETRemoteClientPullDatasetComplete
	refAsReporef := reporef.RefFromDsref(*ref)

	prevRef, err := node.Repo.GetRef(reporef.DatasetRef{Peername: ref.Username, Name: ref.Name})
	if err != nil && err == repo.ErrNotFound {
		if err = node.Repo.PutRef(refAsReporef); err != nil {
			log.Debug(err.Error())
			return nil, fmt.Errorf("error putting dataset in repo: %s", err.Error())
		}

		return dsfs.LoadDataset(ctx, node.Repo.Filesystem(), ref.Path)
	}
	if err != nil {
		return nil, err
	}

	prevRef.Dataset, err = dsfs.LoadDataset(ctx, node.Repo.Filesystem(), prevRef.Path)
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading repo dataset: %s", prevRef.Path)
	}

	ds, err = dsfs.LoadDataset(ctx, node.Repo.Filesystem(), ref.Path)
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading added dataset: %s", ref.Path)
	}
	refAsReporef.Dataset = ds

	if err := base.ReplaceRefIfMoreRecent(node.Repo, &prevRef, &refAsReporef); err != nil {
		return nil, err
	}

	return ds, nil
}

// pullLogs fetches logbook data from a remote & stores it locally
func (c *client) pullLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	log.Debugf("client.pullLogs ref=%q remoteAddr=%q", ref, remoteAddr)
	if t := addressType(remoteAddr); t == "http" {
		remoteAddr = remoteAddr + "/remote/logsync"
	}
	pull, err := c.logsync.NewPull(ref, remoteAddr)
	if err != nil {
		return err
	}

	pull.Merge = true
	_, err = pull.Do(ctx)
	return err
}

// pullDatasetVersion fetches a dataset from a remote source
func (c *client) pullDatasetVersion(ctx context.Context, ref *dsref.Ref, remoteAddr string) error {
	log.Debugf("client.pulldatasetVersion: ref=%q remoteAddr=%q", ref, remoteAddr)

	if ref.Path == "" {
		if _, err := c.NewRemoteRefResolver(remoteAddr).ResolveRef(ctx, ref); err != nil {
			log.Errorf("resolving head ref: %s", err.Error())
			return err
		}
	}

	params, err := sigParams(c.pk, c.profile.Peername, *ref)
	if err != nil {
		log.Debugf("generating sig params error=%q ", err)
		return err
	}

	if t := addressType(remoteAddr); t == "http" {
		remoteAddr = remoteAddr + "/remote/dsync"
	}

	pull, err := c.ds.NewPull(ref.Path, remoteAddr, params)
	if err != nil {
		log.Debugf("NewPull error=%q", err)
		return err
	}

	go func() {
		updates := pull.Updates()
		for {
			select {
			case update := <-updates:
				go func() {
					prog := event.RemoteEvent{
						Ref:        *ref,
						RemoteAddr: remoteAddr,
						Progress:   update,
					}
					if err := c.events.Publish(ctx, event.ETRemoteClientPullVersionProgress, prog); err != nil {
						log.Error("publishing %q event: %q", event.ETRemoteClientPullVersionProgress, err)
					}
				}()
			case <-ctx.Done():
				return
			}
		}
	}()

	if err := pull.Do(ctx); err != nil {
		return err
	}

	// TODO (b5) - this should be part of dsync, no?
	if pinner, ok := c.node.Repo.Filesystem().Filesystem("ipfs").(qfs.PinningFS); ok {
		if err := pinner.Pin(ctx, ref.Path, true); err != nil {
			return err
		}
	}

	return c.events.Publish(ctx, event.ETRemoteClientPullVersionCompleted, event.RemoteEvent{
		Ref:        *ref,
		RemoteAddr: remoteAddr,
	})
}

// RemoveDataset requests a remote remove logbook data from an address
func (c *client) RemoveDataset(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	log.Debugf("client.RemoveDataset ref=%q remoteAddr=%q", ref, remoteAddr)
	if c == nil {
		return ErrNoRemoteClient
	}

	if t := addressType(remoteAddr); t == "http" {
		remoteAddr = remoteAddr + "/remote/logsync"
	}

	log.Debugf("deleting logs for %s from %s", ref.Alias(), remoteAddr)
	if err := c.logsync.DoRemove(ctx, ref, remoteAddr); err != nil {
		return err
	}

	// TODO (b5) - this is currently only requesting a remove one version, it should
	// be requesting *all* versions be removed, but we have no concise way to express thatx
	if err := c.RemoveDatasetVersion(ctx, ref, remoteAddr); err != nil {
		return err
	}

	return c.events.Publish(ctx, event.ETRemoteClientRemoveDatasetCompleted, event.RemoteEvent{
		Ref:        ref,
		RemoteAddr: remoteAddr,
	})
}

// RemoveDatasetVersion asks a remote to remove a dataset version
func (c *client) RemoveDatasetVersion(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	log.Debugf("client.RemoveDatasetVersion ref=%q remoteAddr=%q", ref, remoteAddr)
	if c == nil {
		return ErrNoRemoteClient
	}

	params, err := sigParams(c.pk, c.profile.Peername, ref)
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

func (c *client) signHTTPRequest(req *http.Request) error {
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

// Done returns a channel that the client will send on when finished closing
func (c *client) Done() <-chan struct{} {
	return c.doneCh
}

// DoneErr gives an error that occurred during the shutdown process
func (c *client) DoneErr() error {
	return c.doneErr
}

// PrintProgressBarsOnPushPull writes progress data to the given writer on
// push & pull. requires the event bus that a remote.Client is publishing on
func PrintProgressBarsOnPushPull(w io.Writer, bus event.Bus) {
	var lock sync.Mutex
	progress := map[string]*pb.ProgressBar{}

	// wire up a subscription to print download progress to streams
	bus.Subscribe(func(_ context.Context, typ event.Type, payload interface{}) error {
		lock.Lock()
		defer lock.Unlock()

		switch typ {
		case event.ETRemoteClientPushVersionProgress:
			if evt, ok := payload.(event.RemoteEvent); ok {
				bar, exists := progress[evt.Ref.String()]
				if !exists {
					bar = pb.New(len(evt.Progress))
					bar.SetWriter(w)
					bar.SetMaxWidth(80)
					bar.Start()
					progress[evt.Ref.String()] = bar
				}
				bar.SetCurrent(int64(evt.Progress.CompletedBlocks()))
			}
		case event.ETRemoteClientPushVersionCompleted:
			if evt, ok := payload.(event.RemoteEvent); ok {
				if bar, exists := progress[evt.Ref.String()]; exists {
					bar.SetCurrent(bar.Total())
					bar.Finish()
					delete(progress, evt.Ref.String())
				}
			}
		case event.ETRemoteClientPullVersionProgress:
			if evt, ok := payload.(event.RemoteEvent); ok {
				bar, exists := progress[evt.Ref.String()]
				if !exists {
					bar = pb.New(len(evt.Progress))
					bar.SetWriter(w)
					bar.SetMaxWidth(80)
					bar.Start()
					progress[evt.Ref.String()] = bar
				}
				bar.SetCurrent(int64(evt.Progress.CompletedBlocks()))
			}
		case event.ETRemoteClientPullVersionCompleted:
			if evt, ok := payload.(event.RemoteEvent); ok {
				if bar, exists := progress[evt.Ref.String()]; exists {
					bar.SetCurrent(bar.Total())
					bar.Finish()
					delete(progress, evt.Ref.String())
				}
			}
		}
		return nil
	},
		event.ETRemoteClientPushVersionProgress,
		event.ETRemoteClientPushVersionCompleted,
		event.ETRemoteClientPullVersionProgress,
		event.ETRemoteClientPullVersionCompleted,
	)
}
