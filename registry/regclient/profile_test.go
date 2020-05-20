package regclient

import (
	"encoding/base64"
	"testing"

	"github.com/qri-io/qri/registry"
)

func TestProfileRequests(t *testing.T) {
	tr, cleanup := NewTestRunner(t)
	defer cleanup()

	client := tr.Client
	input := &registry.Profile{
		Username: "b5",
	}

	pubBytes, err := tr.ClientPrivKey.GetPublic().Bytes()
	if err != nil {
		t.Error(err.Error())
	}

	p := &registry.Profile{
		PublicKey: base64.StdEncoding.EncodeToString(pubBytes),
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
