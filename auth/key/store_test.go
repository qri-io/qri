package key_test

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qri/auth/key"
	testkeys "github.com/qri-io/qri/auth/key/test"
)

func TestLocalStore(t *testing.T) {
	path, err := ioutil.TempDir("", "keys")
	if err != nil {
		t.Fatalf("error creating tmp directory: %s", err.Error())
	}
	t.Logf("store: %s", path)
	// defer os.RemoveAll(path)

	ks, err := key.NewLocalStore(filepath.Join(path, "keystore_test.json"))
	if err != nil {
		t.Fatal(err)
	}

	kd0 := testkeys.GetKeyData(0)

	if err = ks.AddPubKey(peer.ID("this_must_fail"), kd0.PrivKey.GetPublic()); err == nil {
		t.Error("expected adding public key with mismatching ID to fail. got nil")
	} else if !errors.Is(err, key.ErrKeyAndIDMismatch) {
		t.Errorf("mismatched ID error must wrap exported pacakge error, got: %s", err)
	}

	if err = ks.AddPubKey(kd0.PeerID, kd0.PrivKey.GetPublic()); err != nil {
		t.Fatal(err)
	}

	if err = ks.AddPrivKey(kd0.PeerID, kd0.PrivKey); err != nil {
		t.Fatal(err)
	}

	if err = ks.AddPrivKey(peer.ID("this_must_fail"), kd0.PrivKey); err == nil {
		t.Error("expected adding private key with mismatching ID to fail. got nil")
	} else if !errors.Is(err, key.ErrKeyAndIDMismatch) {
		t.Errorf("mismatched ID error must wrap exported pacakge error, got: %s", err)
	}

	golden := "testdata/keystore.json"
	path = filepath.Join(path, "keystore_test.json")
	f1, err := ioutil.ReadFile(golden)
	if err != nil {
		t.Errorf("error reading golden file: %s", err.Error())
	}
	f2, err := ioutil.ReadFile(path)
	if err != nil {
		t.Errorf("error reading written file: %s", err.Error())
	}

	if diff := cmp.Diff(f1, f2); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
