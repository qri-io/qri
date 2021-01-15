package profile

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/key"
	cfgtest "github.com/qri-io/qri/config/test"
)

func TestPutProfileWithAddresses(t *testing.T) {
	pp := &config.ProfilePod{
		ID:       "QmU27VdAEUL5NGM6oB56htTxvHLfcGZgsgxrJTdVr2k4zs",
		Peername: "test_peername",
		Created:  time.Unix(1234567890, 0).In(time.UTC),
		Updated:  time.Unix(1234567890, 0).In(time.UTC),
	}
	pro, err := NewProfile(pp)
	if err != nil {
		t.Errorf("error creating new profile: %s", err.Error())
	}
	pid, _ := peer.IDB58Decode("Qmb9Gy14GuCjrhRSjGJQpf5JkgdEdbZrV81Tz4x3ZDreY3")
	pro.PeerIDs = []peer.ID{
		pid,
	}

	path := filepath.Join(os.TempDir(), "profile")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("error creating tmp directory: %s", err.Error())
	}

	pi0 := cfgtest.GetTestPeerInfo(0)
	ks, err := key.NewMemStore(pid, pi0.PrivKey)
	if err != nil {
		t.Fatal(err)
	}

	ps, err := NewLocalStore(filepath.Join(path, "profiles.json"), &Profile{PrivKey: pi0.PrivKey, Peername: "user"}, ks)
	if err != nil {
		t.Fatal(err)
	}

	err = ps.PutProfile(pro)
	if err != nil {
		t.Errorf("error putting profile: %s", err.Error())
	}

	golden := "testdata/simple.json"
	path = filepath.Join(path, "profiles.json")
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
