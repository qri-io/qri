package profile

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	testkeys "github.com/qri-io/qri/auth/key/test"
)

const base64encoded = "QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1"

func TestIDJSON(t *testing.T) {
	idbytes, err := IDB58MustDecode(base64encoded).MarshalJSON()
	if err != nil {
		t.Error(err.Error())
		return
	}
	expect := []byte(`"QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1"`)
	if !bytes.Equal(idbytes, expect) {
		t.Errorf("byte mistmatch. expected: %s, got: %s", string(expect), string(idbytes))
	}
}

func TestPeerID(t *testing.T) {
	kd := testkeys.GetKeyData(0)

	idStr := kd.EncodedPeerID
	if idStr != "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B" {
		t.Errorf("unexpected value for encoded peerID")
	}

	mistakenDecode := "9tmzz8FC9hjBrY1J9NFFt4gjAzGZWCGrKwB4pcdwuSHC7Y4Y7oPPAkrV48ryPYu"

	badlyConstructedProfileID := ID(idStr)
	if badlyConstructedProfileID.Encode() != mistakenDecode {
		t.Errorf("unexpected value for encoded peerID, got %s", badlyConstructedProfileID)
	}

	wellformedProfileID0 := IDB58DecodeOrEmpty(idStr)
	if wellformedProfileID0.Encode() != idStr {
		t.Errorf("unexpected value for encoded peerID, got %s", wellformedProfileID0)
	}

	wellformedProfileID1 := IDFromPeerID(kd.PeerID)
	if wellformedProfileID1.Encode() != idStr {
		t.Errorf("unexpected value for encoded peerID, got %s", wellformedProfileID1)
	}

	wellformedProfileID2 := IDRawByteString(string(kd.PeerID))
	if wellformedProfileID2.Encode() != idStr {
		t.Errorf("unexpected value for encoded peerID, got %s", wellformedProfileID2)
	}
}

func TestStringifyVsEncode(t *testing.T) {
	var pid ID

	kd := testkeys.GetKeyData(0)
	pid = IDFromPeerID(kd.PeerID)

	actual := pid.String()
	expect := `profile.ID{1220ed925aab9128c318acffebcc9a2b66876cebefb2511d2b1e3256f74ce96549a8}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("profileID.String() (-want +got):\n%s", diff)
	}

	actual = pid.Encode()
	expect = kd.EncodedPeerID
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("profileID.String() (-want +got):\n%s", diff)
	}
}

func TestEmpty(t *testing.T) {
	var pid ID
	if !pid.Empty() {
		t.Fatal("expected: profileID.Empty should be true")
	}

	kd := testkeys.GetKeyData(0)
	pid = IDFromPeerID(kd.PeerID)
	if pid.Empty() {
		t.Error("expected: profileID.Empty should be false")
	}
}

func TestMarshalJSON(t *testing.T) {
	var pid ID

	kd := testkeys.GetKeyData(0)
	pid = IDFromPeerID(kd.PeerID)

	data, err := pid.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	actual := string(data)
	expect := fmt.Sprintf("%q", kd.EncodedPeerID)
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("profileID.String() (-want +got):\n%s", diff)
	}
}

func TestMarshalYAML(t *testing.T) {
	var pid ID

	kd := testkeys.GetKeyData(0)
	pid = IDFromPeerID(kd.PeerID)

	actual, err := pid.MarshalYAML()
	if err != nil {
		t.Fatal(err)
	}
	expect := kd.EncodedPeerID
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("profileID.String() (-want +got):\n%s", diff)
	}
}

func TestValidate(t *testing.T) {
	var pid ID

	// profile.ID is empty, Validate will fail
	err := pid.Validate()
	if err == nil {
		t.Fatal("error expected, did not get one")
	}
	expectErr := `empty peer ID`
	if expectErr != err.Error() {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err)
	}

	// profile.ID is invalid, Validate will fail
	// NOTE: Do not call this function this way! Only used here to demonstrate
	// what *not* to do
	pid = IDRawByteString("bad")
	err = pid.Validate()
	if err == nil {
		t.Fatal("error expected, did not get one")
	}
	expectErr = `profile.ID invalid, encodes to "a3c7"`
	if expectErr != err.Error() {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err)
	}

	// profile.ID is invalid due to double-encoding, Validate will fail
	// NOTE: Do not call this function this way! Only used here to demonstrate
	// what *not* to do
	kd := testkeys.GetKeyData(0)
	pid = IDRawByteString(kd.EncodedPeerID)
	err = pid.Validate()
	if err == nil {
		t.Fatal("error expected, did not get one")
	}
	expectErr = `profile.ID invalid, was double encoded as "9tmzz8FC9hjBrY1J9NFFt4gjAzGZWCGrKwB4pcdwuSHC7Y4Y7oPPAkrV48ryPYu". do not pass a base64 encoded string, instead use IDB58Decode(b64encodedID)`
	if expectErr != err.Error() {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err)
	}

	// profile.ID is correctly decoded, Validate will not return an error
	pid, err = IDB58Decode(kd.EncodedPeerID)
	if err != nil {
		t.Fatalf("Decode got error: %s", err)
	}
	err = pid.Validate()
	if err != nil {
		t.Errorf("Validate() got error: %s", err)
	}

	// profile.ID is correctly coerced from peer.ID, Validate will not return an error
	pid = IDFromPeerID(kd.PeerID)
	err = pid.Validate()
	if err != nil {
		t.Errorf("Validate() got error: %s", err)
	}
}
