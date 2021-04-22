package lib

import (
	"context"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"path/filepath"
	"testing"

	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/registry/regserver"
	repotest "github.com/qri-io/qri/repo/test"
)

// Test that running prove sets the profileID for the user
func TestProveProfileKey(t *testing.T) {
	tr := newTestRunner(t)
	defer tr.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	reg, cleanup, err := regserver.NewTempRegistry(ctx, "temp_registry", "", repotest.NewTestCrypto())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	defer cancel()

	// Create a mock registry, point our test runner to its URL
	regClient, _ := regserver.NewMockServerRegistry(*reg)
	tr.Instance.registry = regClient

	// Get an example peer, and add it to the local profile store
	keyData := testkeys.GetKeyData(2)
	pro := &profile.Profile{
		Peername: "test_peer",
		PubKey:   keyData.PrivKey.GetPublic(),
		PrivKey:  keyData.PrivKey,
		ID:       profile.IDFromPeerID(keyData.PeerID),
	}
	repo := tr.Instance.Repo()
	pstore := repo.Profiles()
	err = pstore.SetOwner(pro)
	if err != nil {
		t.Fatal(err)
	}

	// Set path for config file so that it will serialize to disk
	configPath := filepath.Join(tr.TmpDir, "qri", "config.yaml")
	tr.Instance.GetConfig().SetPath(configPath)

	// Call the endpoint to prove our account
	methods := tr.Instance.Registry()
	p := RegistryProfile{
		Username: pro.Peername,
		Email:    "test_peer@qri.io",
		Password: "hunter2",
	}
	if err = methods.ProveProfileKey(tr.Ctx, &RegistryProfileParams{Profile: &p}); err != nil {
		t.Error(err)
	}

	// Peer 3 is used by the mock regserver, it is now used by this peer
	expectProfileID := profile.IDFromPeerID(testkeys.GetKeyData(3).PeerID)
	if pro.ID != expectProfileID {
		t.Errorf("bad profileID for peer after prove. expect: %s, got: %s", expectProfileID, pro.ID)
	}

	// Read config to ensure it contains the expected values
	contents, err := ioutil.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfgData := config.Config{}
	err = yaml.Unmarshal(contents, &cfgData)
	if err != nil {
		t.Fatal(err)
	}

	// Key created by the test runner (this current node's profile) is testKey[0], it stays the
	// same after running prove
	if cfgData.Profile.KeyID != testkeys.GetKeyData(0).EncodedPeerID {
		t.Errorf("profile's keyID should be testKey[0]")
	}
	// ProfileID given to us by registry, by running prove, is testKey[3]
	if cfgData.Profile.ID != testkeys.GetKeyData(3).EncodedPeerID {
		t.Errorf("profile's profileID given by prove command should be testKey[3]")
	}
}
