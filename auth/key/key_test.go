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
