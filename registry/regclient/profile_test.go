package regclient

import (
	"testing"

	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/registry"
)

func TestProfileRequests(t *testing.T) {
	tr, cleanup := NewTestRunner(t)
	defer cleanup()

	client := tr.Client
	input := &registry.Profile{
		Username: "b5",
	}

	pubKeyStr, err := key.EncodePubKeyB64(tr.ClientPrivKey.GetPublic())
	if err != nil {
		t.Error(err.Error())
	}

	p := &registry.Profile{
		PublicKey: pubKeyStr,
	}

	err = client.GetProfile(p)
	if err == nil {
		t.Errorf("expected empty get to error")
	} else if err.Error() != "registry: " {
		t.Errorf("error mistmatch. expected: %s, got: %s", "error 404: ", err.Error())
	}

	_, err = client.PutProfile(input, tr.ClientPrivKey)
	if err != nil {
		t.Error(err.Error())
	}

	err = client.GetProfile(p)
	if err != nil {
		t.Error(err)
	}
	if p.Username != input.Username {
		t.Errorf("expected username to equal %s, got: %s", input.Username, p.Username)
	}

	err = client.DeleteProfile(input, tr.ClientPrivKey)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestRegistryProfileIDGenerator(t *testing.T) {
	gen := key.NewCryptoGenerator()
	pks, pid := gen.GeneratePrivateKeyAndPeerID()
	pk, err := key.DecodeB64PrivKey(pks)
	if err != nil {
		t.Fatal(err)
	}
	pro, err := registry.ProfileFromPrivateKey(&registry.Profile{Username: "test_user"}, pk)
	if err != nil {
		t.Error(err.Error())
	}
	if pid != pro.ProfileID {
		t.Errorf("expected profile IDs to be equal %s, got: %s", pid, pro.ProfileID)
	}
}
