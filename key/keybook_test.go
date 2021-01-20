package key

import (
	"testing"

	cfgtest "github.com/qri-io/qri/config/test"
)

func TestPublicKey(t *testing.T) {
	kb := newKeyBook()
	pi0 := cfgtest.GetTestPeerInfo(0)
	k0 := ID("key_id_0")

	err := kb.AddPubKey(k0, pi0.PubKey)
	if err != nil {
		t.Fatal(err)
	}

	pi1 := cfgtest.GetTestPeerInfo(1)
	k1 := ID("key_id_1")
	err = kb.AddPubKey(k1, pi1.PubKey)
	if err != nil {
		t.Fatal(err)
	}

	tPub := kb.PubKey(k0)
	if tPub != pi0.PubKey {
		t.Fatalf("returned key does not match")
	}

	tPub = kb.PubKey(k1)
	if tPub != pi1.PubKey {
		t.Fatalf("returned key does not match")
	}
}

func TestPrivateKey(t *testing.T) {
	kb := newKeyBook()
	pi0 := cfgtest.GetTestPeerInfo(0)
	k0 := ID("key_id_0")

	err := kb.AddPrivKey(k0, pi0.PrivKey)
	if err != nil {
		t.Fatal(err)
	}

	pi1 := cfgtest.GetTestPeerInfo(1)
	k1 := ID("key_id_1")
	err = kb.AddPrivKey(k1, pi1.PrivKey)
	if err != nil {
		t.Fatal(err)
	}

	tPriv := kb.PrivKey(k0)
	if tPriv != pi0.PrivKey {
		t.Fatalf("returned key does not match")
	}

	tPriv = kb.PrivKey(k1)
	if tPriv != pi1.PrivKey {
		t.Fatalf("returned key does not match")
	}
}

func TestIDsWithKeys(t *testing.T) {
	kb := newKeyBook()
	pi0 := cfgtest.GetTestPeerInfo(0)
	k0 := ID("key_id_0")

	err := kb.AddPrivKey(k0, pi0.PrivKey)
	if err != nil {
		t.Fatal(err)
	}

	pi1 := cfgtest.GetTestPeerInfo(1)
	k1 := ID("key_id_1")
	err = kb.AddPubKey(k1, pi1.PubKey)
	if err != nil {
		t.Fatal(err)
	}

	pids := kb.IDsWithKeys()

	if len(pids) != 2 {
		t.Fatalf("expected to get 2 ids but got: %d", len(pids))
	}

	// the output of kb.IDsWithKeys is in a non-deterministic order
	// so we have to account for all permutations
	if !(pids[0] == k0 && pids[1] == k1) && !(pids[0] == k1 && pids[1] == k0) {
		t.Fatalf("invalid ids returned")
	}
}
