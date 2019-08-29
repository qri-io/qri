package remote

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/multiformats/go-multihash"
	"github.com/qri-io/dag/dsync"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// ErrNoRemoteClient is returned when no client is allocated
var ErrNoRemoteClient = fmt.Errorf("not configured to make remote requests")

// Address extracts the address of a remote from a configuration for a given
// remote name
func Address(cfg *config.Config, name string) (addr string, err error) {
	if name == "" {
		if cfg.Registry.Location != "" {
			return cfg.Registry.Location, nil
		}
		return "", fmt.Errorf("no registry specifiied to use as default remote")
	}

	if dst, found := cfg.Remotes.Get(name); found {
		return dst, nil
	}

	return "", fmt.Errorf(`remote name "%s" not found`, name)
}

// Client issues requests to a remote
type Client struct {
	pk crypto.PrivKey
	ds *dsync.Dsync
}

// NewClient creates a client
func NewClient(node *p2p.QriNode) (*Client, error) {
	capi, err := node.IPFSCoreAPI()
	if err != nil {
		return nil, err
	}

	lng, err := dsync.NewLocalNodeGetter(capi)
	if err != nil {
		return nil, err
	}

	ds, err := dsync.New(lng, capi.Block(), func(dsyncConfig *dsync.Config) {
		if host := node.Host(); host != nil {
			dsyncConfig.Libp2pHost = host
		}

		dsyncConfig.PinAPI = capi.Pin()
	})

	return &Client{
		pk: node.Repo.PrivateKey(),
		ds: ds,
	}, nil
}

// PushDataset pushes the contents of a dataset to a remote
func (c *Client) PushDataset(ctx context.Context, ref repo.DatasetRef, remoteAddr string) error {
	if c == nil {
		return ErrNoRemoteClient
	}
	log.Debugf("pushing dataset %s to %s", ref.Path, remoteAddr)
	push, err := c.ds.NewPush(ref.Path, remoteAddr+"/remote/dsync", true)
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
func (c *Client) PullDataset(ctx context.Context, ref *repo.DatasetRef, remoteAddr string) error {
	if c == nil {
		return ErrNoRemoteClient
	}

	if ref.Path == "" {
		if err := c.ResolveHeadRef(ctx, ref, remoteAddr); err != nil {
			return err
		}
	}

	params, err := sigParams(c.pk, *ref)
	if err != nil {
		return err
	}

	pull, err := c.ds.NewPull(ref.Path, remoteAddr+"/remote/dsync", params)
	if err != nil {
		return err
	}

	return pull.Do(ctx)
}

// RemoveDataset asks a remote to remove a dataset
func (c *Client) RemoveDataset(ctx context.Context, ref repo.DatasetRef, remoteAddr string) error {
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
func (c *Client) ResolveHeadRef(ctx context.Context, ref *repo.DatasetRef, remoteAddr string) error {
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

func resolveHeadRefHTTP(ctx context.Context, ref *repo.DatasetRef, remoteAddr string) error {
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

func sigParams(pk crypto.PrivKey, ref repo.DatasetRef) (map[string]string, error) {

	pid, err := calcProfileID(pk)
	if err != nil {
		return nil, err
	}

	now := fmt.Sprintf("%d", time.Now().In(time.UTC).Unix())
	rss := requestSigningString(now, pid, ref.Path)

	b64Sig, err := signString(pk, rss)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"peername":  ref.Peername,
		"name":      ref.Name,
		"profileId": pid,
		"path":      ref.Path,

		"timestamp": now,
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

func calcProfileID(privKey crypto.PrivKey) (string, error) {
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

func addressType(remoteAddr string) string {
	// if a valid base58 peerID is passed, we're doing a p2p dsync
	if _, err := peer.IDB58Decode(remoteAddr); err == nil {
		return "p2p"
	} else if strings.HasPrefix(remoteAddr, "http") {
		return "http"
	}

	return ""
}
