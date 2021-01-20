package key

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	cfgtest "github.com/qri-io/qri/config/test"
)

func TestLocalStore(t *testing.T) {
	path := filepath.Join(os.TempDir(), "keys")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("error creating tmp directory: %s", err.Error())
	}

	ks, err := NewLocalStore(filepath.Join(path, "keystore_test.json"))
	if err != nil {
		t.Fatal(err)
	}

	pi0 := cfgtest.GetTestPeerInfo(0)
	k0 := ID("key_id_0")
	err = ks.AddPubKey(k0, pi0.PubKey)
	if err != nil {
		t.Fatal(err)
	}

	err = ks.AddPrivKey(k0, pi0.PrivKey)
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
