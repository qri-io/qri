package key_test

import (
	"testing"

	"github.com/qri-io/qri/auth/key"
	testkeys "github.com/qri-io/qri/auth/key/test"
)

func TestKeyDecodeID(t *testing.T) {
	kd0 := testkeys.GetKeyData(0)

	id, err := key.DecodeID(kd0.EncodedPeerID)
	if err != nil {
		t.Fatal(err)
	}

	if kd0.EncodedPeerID != id.String() {
		t.Errorf("string mistmatch want: %q got: %q", kd0.EncodedPeerID, id.String())
	}
}

func TestIDFromPriv(t *testing.T) {
	kd := testkeys.GetKeyData(0)
	expect := kd.KeyID.String()
	got, err := key.IDFromPrivKey(kd.PrivKey)
	if err != nil {
		t.Error(err)
	}

	if expect != got {
		t.Errorf("ID mismatch. expected: '%s', got: '%s'", expect, got)
	}
}

func TestIDFromPub(t *testing.T) {
	if _, err := key.IDFromPubKey(nil); err == nil {
		t.Errorf("expected error calculating the ID of nil")
	}

	kd := testkeys.GetKeyData(1)
	expect := kd.KeyID.String()
	got, err := key.IDFromPubKey(kd.PrivKey.GetPublic())
	if err != nil {
		t.Error(err)
	}

	if expect != got {
		t.Errorf("ID mismatch. expected: '%s', got: '%s'", expect, got)
	}
}
