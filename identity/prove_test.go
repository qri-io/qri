package identity

import (
	"testing"
	"time"

	"github.com/qri-io/qri/config/test"
)

func TestCreateKeyJoinToken(t *testing.T) {
	prevTS := Timestamp
	defer func() { Timestamp = prevTS }()
	Timestamp = func() time.Time { return time.Time{} }

	registry := test.GetTestPeerInfo(0)
	a := test.GetTestPeerInfo(1).PubKey
	b := test.GetTestPeerInfo(2).PubKey

	token, err := CreateKeyJoinToken(registry.PrivKey, a, b)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(token.Raw)

}
