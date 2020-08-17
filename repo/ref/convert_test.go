package reporef

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	testPeers "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo/profile"
)

func TestRefFromDsref(t *testing.T) {
	d := dsref.Ref{
		Username:  "test_peer_ref_from_dsref",
		ProfileID: "",
		Name:      "my_ds",
		Path:      "/mem/QmExaMpLe",
	}

	ref := RefFromDsref(d)
	expectRef := DatasetRef{
		Peername:  "test_peer_ref_from_dsref",
		ProfileID: profile.IDRawByteString(""),
		Name:      "my_ds",
		Path:      "/mem/QmExaMpLe",
	}
	if diff := cmp.Diff(expectRef, ref); diff != "" {
		t.Errorf("mismatch (-want, +got)\ndiff:%v", diff)
	}
}

func TestRefFromDsrefCantDecode(t *testing.T) {
	d := dsref.Ref{
		Username:  "a_user",
		ProfileID: "testProfileID",
		Name:      "some_name",
		Path:      "/mem/QmExaMpLe2",
	}

	ref := RefFromDsref(d)
	expectRef := DatasetRef{
		Peername:  "a_user",
		ProfileID: profile.IDRawByteString(""),
		Name:      "some_name",
		Path:      "/mem/QmExaMpLe2",
	}
	if diff := cmp.Diff(expectRef, ref); diff != "" {
		t.Errorf("mismatch (-want, +got)\ndiff:%v", diff)
	}
}

func TestRefFromDsrefCorrectProfileID(t *testing.T) {
	info := testPeers.GetTestPeerInfo(0)

	d := dsref.Ref{
		Username:  "someone",
		ProfileID: info.EncodedPeerID,
		Name:      "example",
		Path:      "/mem/QmExaMpLe3",
	}

	ref := RefFromDsref(d)
	expectRef := DatasetRef{
		Peername:  "someone",
		ProfileID: profile.IDFromPeerID(info.PeerID),
		Name:      "example",
		Path:      "/mem/QmExaMpLe3",
	}
	if diff := cmp.Diff(expectRef, ref); diff != "" {
		t.Errorf("mismatch (-want, +got)\ndiff:%v", diff)
	}
}
