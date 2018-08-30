package config

import (
	"reflect"
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

	p = DefaultP2PForTesting()
	_, err = p.DecodePrivateKey()
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
}

func TestP2PValidate(t *testing.T) {
	err := DefaultP2PForTesting().Validate()
	if err != nil {
		t.Errorf("error validating default p2p: %s", err)
	}
}

func TestP2PCopy(t *testing.T) {
	cases := []struct {
		p2p *P2P
	}{
		{DefaultP2PForTesting()},
	}
	for i, c := range cases {
		cpy := c.p2p.Copy()
		if !reflect.DeepEqual(cpy, c.p2p) {
			t.Errorf("P2P Copy test case %v, p2p structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.p2p)
			continue
		}
		cpy.QriBootstrapAddrs[0] = ""
		if reflect.DeepEqual(cpy, c.p2p) {
			t.Errorf("P2P Copy test case %v, editing one p2p struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.p2p)
			continue
		}
	}
}
