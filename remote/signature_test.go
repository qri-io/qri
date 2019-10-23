package remote

import (
	"testing"

	"github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestVerifySigParams(t *testing.T) {
	peerInfo := test.GetTestPeerInfo(0)
	pid, err := calcProfileID(peerInfo.PrivKey)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	profileID, err := profile.NewB58ID(pid)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	ref := repo.DatasetRef{
		Path:      "foo",
		Peername:  "bar",
		Name:      "baz",
		ProfileID: profileID,
	}
	sigParams, err := sigParams(peerInfo.PrivKey, ref)
	if err != nil {
		panic(err)
	}

	verified, err := VerifySigParams(peerInfo.PubKey, sigParams)
	if err != nil {
		t.Errorf("case 'should verify', expected no error, got '%s'", err)
	}
	if verified == false {
		t.Errorf("case 'should verify', expected verification to be true, was false")
	}
}
