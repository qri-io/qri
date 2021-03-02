package key_test

import (
	"testing"

	"github.com/qri-io/qri/auth/key"
	testkeys "github.com/qri-io/qri/auth/key/test"
)

func TestPublicKey(t *testing.T) {
	kb, err := key.NewMemStore()
	if err != nil {
		t.Fatal(err)
	}

	ki0 := testkeys.GetKeyData(0)
	k0 := ki0.PrivKey.GetPublic()
	k0AltID := key.ID("key_id_0")

	if err = kb.AddPubKey(k0AltID, k0); err != nil {
		t.Fatal(err)
	}

	ki1 := testkeys.GetKeyData(1)
	k1 := ki1.PrivKey.GetPublic()
	k1AltID := key.ID("key_id_1")
	err = kb.AddPubKey(k1AltID, k1)
	if err != nil {
		t.Fatal(err)
	}

	tPub := kb.PubKey(k0AltID)
	if tPub != k0 {
		t.Fatalf("returned key does not match")
	}

	tPub = kb.PubKey(k1AltID)
	if tPub != k1 {
		t.Fatalf("returned key does not match")
	}
}

func TestPrivateKey(t *testing.T) {
	kb, err := key.NewMemStore()
	if err != nil {
		t.Fatal(err)
	}

	kd0 := testkeys.GetKeyData(0)
	k0AltID := key.ID("key_id_0")

	if err := kb.AddPrivKey(k0AltID, kd0.PrivKey); err != nil {
		t.Fatal(err)
	}

	kd1 := testkeys.GetKeyData(1)
	k1AltID := key.ID("key_id_1")
	err = kb.AddPrivKey(k1AltID, kd1.PrivKey)
	if err != nil {
		t.Fatal(err)
	}

	tPriv := kb.PrivKey(k0AltID)
	if tPriv != kd0.PrivKey {
		t.Fatalf("returned key does not match")
	}

	tPriv = kb.PrivKey(k1AltID)
	if tPriv != kd1.PrivKey {
		t.Fatalf("returned key does not match")
	}
}

func TestIDsWithKeys(t *testing.T) {
	kb, err := key.NewMemStore()
	if err != nil {
		t.Fatal(err)
	}

	kd0 := testkeys.GetKeyData(0)
	if err = kb.AddPrivKey(kd0.KeyID, kd0.PrivKey); err != nil {
		t.Fatal(err)
	}

	kd1 := testkeys.GetKeyData(1)
	err = kb.AddPubKey(kd1.KeyID, kd1.PrivKey.GetPublic())
	if err != nil {
		t.Fatal(err)
	}

	pids := kb.IDsWithKeys()

	if len(pids) != 2 {
		t.Fatalf("expected to get 2 ids but got: %d", len(pids))
	}

	// the output of kb.IDsWithKeys is in a non-deterministic order
	// so we have to account for all permutations
	if !(pids[0] == kd0.KeyID && pids[1] == kd1.KeyID) && !(pids[0] == kd1.KeyID && pids[1] == kd0.KeyID) {
		t.Fatalf("invalid ids returned")
	}
}
