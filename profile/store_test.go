package profile

import (
	"encoding/json"
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
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/config"
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

	kd0 := testkeys.GetKeyData(0)

	ks, err := key.NewMemStore()
	if err != nil {
		t.Fatal(err)
	}

	owner := &Profile{
		ID:       ID(kd0.PeerID),
		Peername: "user",
		PrivKey:  kd0.PrivKey,
	}
	ps, err := NewLocalStore(filepath.Join(path, "profiles.json"), owner, ks)
	if err != nil {
		t.Fatal(err)
	}

	err = ps.PutProfile(pro)
	if err != nil {
		t.Errorf("error putting profile: %s", err.Error())
	}

	goldenFilepath := "testdata/simple.json"
	gf, err := ioutil.ReadFile(goldenFilepath)
	if err != nil {
		t.Errorf("error reading golden file: %s", err.Error())
	}
	golden := map[string]interface{}{}
	if err := json.Unmarshal(gf, &golden); err != nil {
		t.Fatal(err)
	}

	path = filepath.Join(path, "profiles.json")
	f, err := ioutil.ReadFile(path)
	if err != nil {
		t.Errorf("error reading written file: %s", err.Error())
	}
	got := map[string]interface{}{}
	if err := json.Unmarshal(f, &got); err != nil {
		t.Fatal(err)
	}

	t.Log(string(f))
	if diff := cmp.Diff(golden, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestProfilesWithKeys(t *testing.T) {
	kd0 := testkeys.GetKeyData(0)

	ks, err := key.NewMemStore()
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(os.TempDir(), "profile_keys")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("error creating tmp directory: %s", err.Error())
	}

	owner := &Profile{
		ID:       ID(kd0.PeerID),
		Peername: "user",
		PrivKey:  kd0.PrivKey,
	}
	ps, err := NewLocalStore(filepath.Join(path, "profiles.json"), owner, ks)
	if err != nil {
		t.Fatal(err)
	}

	pp := &config.ProfilePod{
		ID:       kd0.PeerID.String(),
		Peername: "p0",
		Created:  time.Unix(1234567890, 0).In(time.UTC),
		Updated:  time.Unix(1234567890, 0).In(time.UTC),
	}
	pro, err := NewProfile(pp)
	if err != nil {
		t.Fatal(err)
	}

	pro.PrivKey = kd0.PrivKey
	pro.PubKey = kd0.PrivKey.GetPublic()

	err = ps.PutProfile(pro)
	if err != nil {
		t.Fatal(err)
	}

	tPro, err := ps.GetProfile(pro.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !tPro.PrivKey.Equals(kd0.PrivKey) {
		t.Fatalf("private keys don't match\ngot:  %#v\nwant: %#v", tPro.PrivKey, kd0.PrivKey.GetPublic())
	}

	if !tPro.PubKey.Equals(kd0.PrivKey.GetPublic()) {
		t.Fatalf("public keys don't match.\ngot:  %#v\nwant: %#v", tPro.PubKey, kd0.PrivKey.GetPublic())
	}
}

func TestMemStoreGetOwner(t *testing.T) {
	kd0 := testkeys.GetKeyData(0)
	id := ID(kd0.PeerID)
	owner := &Profile{ID: id, PrivKey: kd0.PrivKey, Peername: "owner"}
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

	if diff := cmp.Diff(owner, pro, cmpopts.IgnoreUnexported(Profile{}, crypto.RsaPublicKey{}, crypto.RsaPrivateKey{}, crypto.ECDSAPublicKey{}, crypto.ECDSAPrivateKey{})); diff != "" {
		t.Errorf("get owner mismatch. (-want +got):\n%s", diff)
	}
}

func TestResolveUsername(t *testing.T) {
	kd0 := testkeys.GetKeyData(0)
	kd1 := testkeys.GetKeyData(1)
	kd2 := testkeys.GetKeyData(2)

	ks, err := key.NewMemStore()
	if err != nil {
		t.Fatal(err)
	}

	owner := &Profile{ID: ID(kd0.PeerID), PrivKey: kd0.PrivKey, Peername: "owner"}
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

	marjorieA := &Profile{ID: ID(kd1.PeerID), PrivKey: kd1.PrivKey, Peername: "marjorie", Email: "marjorie_a@aol.com"}
	marjorieB := &Profile{ID: ID(kd2.PeerID), PrivKey: kd2.PrivKey, Peername: "marjorie", Email: "marjorie_b@aol.com"}

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
