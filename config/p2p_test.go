package config

import (
	"testing"
)

func TestP2PDecodePrivateKey(t *testing.T) {
	missingErr := "missing private key"
	p := &P2P{}
	_, err := p.DecodePrivateKey()
	if err == nil {
		t.Errorf("expected empty private key to err")
	} else if err.Error() != missingErr {
		t.Errorf("error mismatch. expected: %s, got: %s", missingErr, err.Error())
	}

	invalidErr := "decoding private key: illegal base64 data at input byte 4"
	p = &P2P{PrivKey: "invalid"}
	_, err = p.DecodePrivateKey()
	if err == nil {
		t.Errorf("expected empty private key to err")
	} else if err.Error() != invalidErr {
		t.Errorf("error mismatch. expected: %s, got: %s", invalidErr, err.Error())
	}

	// run this test a few times to ensure default profile consistently generates
	// a valid PrivateKey
	for i := 0; i < 10; i++ {
		p = DefaultP2P()
		_, err = p.DecodePrivateKey()
		if err != nil {
			t.Errorf("iter %d unexpected error: %s", i, err.Error())
		}
	}
}

func TestP2PValidate(t *testing.T) {
	err := DefaultP2P().Validate()
	if err != nil {
		t.Errorf("error validating default p2p: %s", err)
	}
}
