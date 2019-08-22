// Package remote implements syncronization between qri instances
package remote

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	golog "github.com/ipfs/go-log"
	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/multiformats/go-multihash"
	"github.com/qri-io/dag/dsync"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
)

var log = golog.Logger("remote")

// Address extracts the address of a remote from a configuration for a given
// remote name
func Address(cfg *config.Config, name string) (addr string, err error) {
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

// PushDataset pushes the contents of a dataset to a remote
func PushDataset(ctx context.Context, dsync *dsync.Dsync, pk crypto.PrivKey, ref repo.DatasetRef, remoteAddr string) error {
	log.Debugf("pushing dataset %s to %s", ref.Path, remoteAddr)
	push, err := dsync.NewPush(ref.Path, remoteAddr, true)
	if err != nil {
		return err
	}

	params, err := sigParams(pk, ref)
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

// RemoveDataset asks a remote to remove a dataset
func RemoveDataset(ctx context.Context, pk crypto.PrivKey, ref repo.DatasetRef, remoteAddr string) error {
	log.Debugf("removing dataset %s from %s", ref.Path, remoteAddr)
	params, err := sigParams(pk, ref)
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
	u.Path = "/remote/datasets"

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
