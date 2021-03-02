package key_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	k0AltID := key.ID("key_id_0")
	err = ks.AddPubKey(k0AltID, kd0.PrivKey.GetPublic())
	if err != nil {
		t.Fatal(err)
	}

	err = ks.AddPrivKey(k0AltID, kd0.PrivKey)
	if err != nil {
		t.Fatal(err)
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
