package fsrepo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo/profile"
	"github.com/sergi/go-diff/diffmatchpatch"

	"gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

func TestPutProfileWithAddresses(t *testing.T) {
	pp := &config.ProfilePod{
		ID:       "QmU27VdAEUL5NGM6oB56htTxvHLfcGZgsgxrJTdVr2k4zs",
		Peername: "test_peername",
		Created:  time.Unix(1234567890, 0).In(time.UTC),
		Updated:  time.Unix(1234567890, 0).In(time.UTC),
	}
	pro, err := profile.NewProfile(pp)
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
	ps := NewProfileStore(basepath(path))
	err = ps.PutProfile(pro)
	if err != nil {
		t.Errorf("error putting profile: %s", err.Error())
	}

	golden := "testdata/simple.json"
	path = filepath.Join(path, "peers.json")
	f1, err := ioutil.ReadFile(golden)
	if err != nil {
		t.Errorf("error reading golden file: %s", err.Error())
	}
	f2, err := ioutil.ReadFile(path)
	if err != nil {
		t.Errorf("error reading written file: %s", err.Error())
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(f1), string(f2), false)
	if len(diffs) > 1 {
		fmt.Println(dmp.DiffPrettyText(diffs))
		t.Errorf("failed to match: %s <> %s", golden, path)
	}
}
