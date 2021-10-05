package remote

import (
	"encoding/base64"
	"fmt"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/dsref"
)

var (
	// nowFunc is an ps function for getting timestamps
	nowFunc = time.Now
)

func sigParams(pk crypto.PrivKey, subjectUsername string, ref dsref.Ref) (map[string]string, error) {
	pid, err := key.IDFromPrivKey(pk)
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
		"username":  ref.Username,
		"peername":  ref.Username,
		"name":      ref.Name,
		"profileID": ref.ProfileID,
		"path":      ref.Path,

		"pid": pid,
		// subject_username is the client node's username, will be used
		// on the remote side to determine access control
		"subject_username": subjectUsername,
		"timestamp":        now,
		"signature":        b64Sig,
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
	_, ok = params["subject_username"]
	if !ok {
		return false, fmt.Errorf("params need key 'subject_username'")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	str := base64.StdEncoding.EncodeToString(sigBytes)
	if str != signature {
		return false, fmt.Errorf("signature was %q, after decode then encode it was %q", signature, str)
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
