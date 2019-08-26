package registry

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multihash"
)

// Profile is a shorthand version of qri-io/qri/repo/profile.Profile
// I'm toying with the idea of using "handle" instead of "peername",
// so it's "handle" here for now
type Profile struct {
	ProfileID string
	Handle    string
	Signature string `json:",omitempty"`
	PublicKey string `json:",omitempty"`
	Created   time.Time
}

// Validate is a sanity check that all required values are present
func (p *Profile) Validate() error {
	if p.Handle == "" {
		return fmt.Errorf("handle is required")
	}
	if p.ProfileID == "" {
		return fmt.Errorf("profileID is required")
	}
	if p.Signature == "" {
		return fmt.Errorf("signature is required")
	}
	if p.PublicKey == "" {
		return fmt.Errorf("publickey is required")
	}
	return nil
}

// Verify checks a profile's proof of key ownership
// Registree's must prove they have control of the private key by signing the desired handle,
// which is validated with a provided public key. Public key, handle, and date of
func (p *Profile) Verify() error {
	return verify(p.PublicKey, p.Signature, []byte(p.Handle))
}

// ProfileFromPrivateKey generates a profile struct from a private key & desired profile handle
// It adds all the necessary components to pass profiles.Register, creating base64-encoded
// PublicKey & Signature, and base58-encoded ProfileID
func ProfileFromPrivateKey(handle string, privKey crypto.PrivKey) (*Profile, error) {
	p := &Profile{
		Handle: handle,
	}

	sigbytes, err := privKey.Sign([]byte(p.Handle))
	if err != nil {
		return nil, fmt.Errorf("error signing %s", err.Error())
	}

	p.Signature = base64.StdEncoding.EncodeToString(sigbytes)

	pubkeybytes, err := privKey.GetPublic().Bytes()
	if err != nil {
		return nil, fmt.Errorf("error getting pubkey bytes: %s", err.Error())
	}

	mh, err := multihash.Sum(pubkeybytes, multihash.SHA2_256, 32)
	if err != nil {
		return nil, fmt.Errorf("error summing pubkey: %s", err.Error())
	}

	p.ProfileID = mh.B58String()
	p.PublicKey = base64.StdEncoding.EncodeToString(pubkeybytes)

	return p, nil
}
