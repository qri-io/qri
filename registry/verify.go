package registry

import (
	"encoding/base64"
	"fmt"

	"github.com/qri-io/qri/auth/key"
)

// verify accepts base64 encoded keys & signatures to validate data
func verify(b64PubKey, b64Signature string, data []byte) error {
	pubkey, err := key.DecodeB64PubKey(b64PubKey)
	if err != nil {
		return err
	}

	sigbytes, err := base64.StdEncoding.DecodeString(b64Signature)
	if err != nil {
		return fmt.Errorf("signature base64 encoding: %s", err.Error())
	}

	valid, err := pubkey.Verify(data, sigbytes)
	if err != nil {
		return fmt.Errorf("invalid signature: %s", err.Error())
	}

	if !valid {
		return fmt.Errorf("mismatched signature")
	}

	return nil
}
