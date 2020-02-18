package profile

import (
	"bytes"
	"testing"

	testPeers "github.com/qri-io/qri/config/test"
)

func TestIDJSON(t *testing.T) {
	idbytes, err := IDB58MustDecode("QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1").MarshalJSON()
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
	peerInfo := testPeers.GetTestPeerInfo(0)

	idStr := peerInfo.EncodedPeerID
	if idStr != "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B" {
		t.Errorf("unexpected value for encoded peerID")
	}

	mistakenDecode := "9tmzz8FC9hjBrY1J9NFFt4gjAzGZWCGrKwB4pcdwuSHC7Y4Y7oPPAkrV48ryPYu"

	badlyConstructedProfileID := ID(idStr)
	if badlyConstructedProfileID.String() != mistakenDecode {
		t.Errorf("unexpected value for encoded peerID, got %s", badlyConstructedProfileID)
	}

	wellformedProfileID0 := IDB58DecodeOrEmpty(idStr)
	if wellformedProfileID0.String() != idStr {
		t.Errorf("unexpected value for encoded peerID, got %s", wellformedProfileID0)
	}

	wellformedProfileID1 := IDFromPeerID(peerInfo.PeerID)
	if wellformedProfileID1.String() != idStr {
		t.Errorf("unexpected value for encoded peerID, got %s", wellformedProfileID1)
	}

	wellformedProfileID2 := IDRawByteString(string(peerInfo.PeerID))
	if wellformedProfileID2.String() != idStr {
		t.Errorf("unexpected value for encoded peerID, got %s", wellformedProfileID2)
	}
}
