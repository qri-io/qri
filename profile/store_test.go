package profile

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/libp2p/go-libp2p-core/peer"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/config"
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

	ks, err := key.NewMemStore()
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

func TestProfilesWithKeys(t *testing.T) {
	pi0 := cfgtest.GetTestPeerInfo(0)

	ks, err := key.NewMemStore()
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(os.TempDir(), "profile_keys")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("error creating tmp directory: %s", err.Error())
	}

	ps, err := NewLocalStore(filepath.Join(path, "profiles.json"), &Profile{PrivKey: pi0.PrivKey, Peername: "user"}, ks)
	if err != nil {
		t.Fatal(err)
	}

	pp := &config.ProfilePod{
		ID:       pi0.PeerID.String(),
		Peername: "p0",
		Created:  time.Unix(1234567890, 0).In(time.UTC),
		Updated:  time.Unix(1234567890, 0).In(time.UTC),
	}
	pro, err := NewProfile(pp)
	if err != nil {
		t.Fatal(err)
	}

	pro.PrivKey = pi0.PrivKey
	pro.PubKey = pi0.PubKey

	err = ps.PutProfile(pro)
	if err != nil {
		t.Fatal(err)
	}

	tPro, err := ps.GetProfile(pro.ID)
	if err != nil {
		t.Fatal(err)
	}

	if tPro.PrivKey != pi0.PrivKey {
		t.Fatalf("keys don't match")
	}

	if tPro.PubKey != pi0.PubKey {
		t.Fatalf("keys don't match")
	}
}

func TestMemStoreGetOwner(t *testing.T) {
	pi0 := cfgtest.GetTestPeerInfo(0)
	id := ID(pi0.PeerID)
	owner := &Profile{ID: id, PrivKey: pi0.PrivKey, Peername: "owner"}
	ks, err := key.NewMemStore()
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewMemStore(owner, ks)
	if err != nil {
		t.Fatal(err)
	}

	pro, err := s.GetProfile(id)
	if err != nil {
		t.Fatal(err)
	}

	if pro.PrivKey == nil {
		t.Error("getting owner profile must return profile with private key populated")
	}

	if diff := cmp.Diff(owner, pro, cmpopts.IgnoreUnexported(Profile{}, crypto.RsaPrivateKey{}, crypto.ECDSAPrivateKey{})); diff != "" {
		t.Errorf("get owner mismatch. (-want +got):\n%s", diff)
	}
}

func TestResolveUsername(t *testing.T) {
	pi0 := cfgtest.GetTestPeerInfo(0)
	pi1 := cfgtest.GetTestPeerInfo(1)
	pi2 := cfgtest.GetTestPeerInfo(2)

	ks, err := key.NewMemStore()
	if err != nil {
		t.Fatal(err)
	}

	owner := &Profile{ID: ID(pi0.PeerID), PrivKey: pi0.PrivKey, Peername: "owner"}
	s, err := NewMemStore(owner, ks)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ResolveUsername(s, "unknown"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected unknown username to return ErrNotFound or wrap of ErrNotFound. got: %#v", err)
	}

	pro, err := ResolveUsername(s, "owner")
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(owner, pro, cmpopts.IgnoreUnexported(Profile{}, crypto.RsaPrivateKey{}, crypto.ECDSAPrivateKey{})); diff != "" {
		t.Errorf("get owner mismatch. (-want +got):\n%s", diff)
	}

	marjorieA := &Profile{ID: ID(pi1.PeerID), PrivKey: pi1.PrivKey, Peername: "marjorie", Email: "marjorie_a@aol.com"}
	marjorieB := &Profile{ID: ID(pi2.PeerID), PrivKey: pi2.PrivKey, Peername: "marjorie", Email: "marjorie_b@aol.com"}

	if err := s.PutProfile(marjorieA); err != nil {
		t.Fatal(err)
	}
	if err := s.PutProfile(marjorieB); err != nil {
		t.Fatal(err)
	}

	if _, err := ResolveUsername(s, "marjorie"); !errors.Is(err, ErrAmbiguousUsername) {
		t.Errorf("expected duplicated username to return ErrAmbiguousUsername or wrap of that error. got: %#v", err)
	}
}
