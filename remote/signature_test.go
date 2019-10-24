package remote

import (
	"testing"

	"github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestVerifySigParams(t *testing.T) {
	peerInfo0 := test.GetTestPeerInfo(0)
	pid, err := calcProfileID(peerInfo0.PrivKey)
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
	sigParams, err := sigParams(peerInfo0.PrivKey, ref)
	if err != nil {
		panic(err)
	}

	verified, err := VerifySigParams(peerInfo0.PubKey, sigParams)
	if err != nil {
		t.Errorf("case 'should verify', expected no error, got '%s'", err)
	}
	if verified == false {
		t.Errorf("case 'should verify', expected verification to be true, but was false")
	}

	peerInfo1 := test.GetTestPeerInfo(1)
	verified, err = VerifySigParams(peerInfo1.PubKey, sigParams)
	if err == nil {
		t.Errorf("case 'should not verify', expected error 'crypto/rsa: verification error', got no error")
	}
	if err != nil && err.Error() != "crypto/rsa: verification error" {
		t.Errorf("case 'should not verify', expected error 'crypto/rsa: verification error', got error, '%s'", err)
	}
	if verified == true {
		t.Errorf("case 'should not verify', expected verification to be false, but was true")
	}
}
