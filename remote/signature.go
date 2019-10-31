package remote

import (
	"encoding/base64"
	"fmt"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multihash"
	"github.com/qri-io/qri/repo"
)

var (
	// nowFunc is an ps function for getting timestamps
	nowFunc = time.Now
)

func sigParams(pk crypto.PrivKey, ref repo.DatasetRef) (map[string]string, error) {
	pid, err := calcProfileID(pk)
	if err != nil {
		return nil, err
	}

	now := fmt.Sprintf("%d", nowFunc().In(time.UTC).Unix())
	rss := requestSigningString(now, pid, ref.Path)
	b64Sig, err := signString(pk, rss)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"peername":  ref.Peername,
		"name":      ref.Name,
		"profileID": ref.ProfileID.String(),
		"path":      ref.Path,

		"pid":       pid,
		"timestamp": now,
		"signature": b64Sig,
	}, nil
}

// VerifySigParams takes a public key and a map[string]string params and verifies
// the the signature is correct
// TODO (ramfox): should be refactored to be private once remotes have their
// own keystore and can make the replation between a pid and a public key
// on their own
func VerifySigParams(pubkey crypto.PubKey, params map[string]string) (bool, error) {
	timestamp, ok := params["timestamp"]
	if !ok {
		return false, fmt.Errorf("params need key 'timestamp'")
	}
	pid, ok := params["pid"]
	if !ok {
		return false, fmt.Errorf("params need key 'pid'")
	}
	path, ok := params["path"]
	if !ok {
		return false, fmt.Errorf("params need key 'path'")
	}
	signature, ok := params["signature"]
	if !ok {
		return false, fmt.Errorf("params need key 'signature'")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	str := base64.StdEncoding.EncodeToString(sigBytes)
	if str != signature {
		return false, fmt.Errorf("signature was '%s', after decode then encode it was '%s", signature, str)
	}
	rss := requestSigningString(timestamp, pid, path)
	return pubkey.Verify([]byte(rss), sigBytes)
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
