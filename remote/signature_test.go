package remote

import (
	"testing"

	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/profile"
)

func TestVerifySigParams(t *testing.T) {
	kd0 := testkeys.GetKeyData(0)
	pid, err := calcProfileID(kd0.PrivKey)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	profileID, err := profile.IDB58Decode(pid)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	ref := dsref.Ref{
		Path:      "foo",
		Username:  "bar",
		Name:      "baz",
		ProfileID: profileID.Encode(),
	}
	sigParams, err := sigParams(kd0.PrivKey, "bar", ref)
	if err != nil {
		panic(err)
	}

	verified, err := VerifySigParams(kd0.PrivKey.GetPublic(), sigParams)
	if err != nil {
		t.Errorf("case 'should verify', expected no error, got '%s'", err)
	}
	if verified == false {
		t.Errorf("case 'should verify', expected verification to be true, but was false")
	}

	kd1 := testkeys.GetKeyData(1)
	verified, err = VerifySigParams(kd1.PrivKey.GetPublic(), sigParams)
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
