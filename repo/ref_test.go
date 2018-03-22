package repo

import (
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo/profile"
)

var cases = []struct {
	ref         DatasetRef
	String      string
	AliasString string
}{
	{DatasetRef{
		Peername: "peername",
	}, "peername", "peername"},
	{DatasetRef{
		Peername: "peername",
		Name:     "datasetname",
	}, "peername/datasetname", "peername/datasetname"},

	{DatasetRef{
		PeerID: "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}, "@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", ""},
	{DatasetRef{
		Path: "/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1",
	}, "@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", ""},

	{DatasetRef{
		Peername: "peername",
		Name:     "datasetname",
		PeerID:   "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}, "peername/datasetname@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", "peername/datasetname"},
	{DatasetRef{
		Peername: "peername",
		Name:     "datasetname",
		PeerID:   "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
		Path:     "/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1",
	}, "peername/datasetname@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "peername/datasetname"},

	{DatasetRef{
		PeerID:   "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
		Peername: "lucille",
	}, "lucille@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", "lucille"},
	{DatasetRef{
		PeerID:   "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
		Peername: "lucille",
		Name:     "ball",
		Path:     "/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1",
	}, "lucille/ball@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "lucille/ball"},

	{DatasetRef{
		PeerID:   "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
		Peername: "bad_name",
	}, "bad_name@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", "bad_name"},
	{DatasetRef{
		PeerID:   "badID",
		Peername: "me",
	}, "me@badID", "me"},
}

func TestDatasetRefString(t *testing.T) {
	for i, c := range cases {
		if c.ref.String() != c.String {
			t.Errorf("case %d:\n%s\n%s", i, c.ref.String(), c.String)
			continue
		}
	}
}

func TestDatasetRefAliasString(t *testing.T) {
	for i, c := range cases {
		if c.ref.AliasString() != c.AliasString {
			t.Errorf("case %d:\n%s\n%s", i, c.ref.AliasString(), c.AliasString)
			continue
		}
	}
}

func TestParseDatasetRef(t *testing.T) {
	peernameDatasetRef := DatasetRef{
		Peername: "peername",
	}

	nameDatasetRef := DatasetRef{
		Peername: "peername",
		Name:     "datasetname",
	}

	peerIDDatasetRef := DatasetRef{
		PeerID: "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	idNameDatasetRef := DatasetRef{
		PeerID: "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
		Name:   "datasetname",
	}

	idFullDatasetRef := DatasetRef{
		PeerID: "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
		Name:   "datasetname",
		Path:   "/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	idFullIPFSDatasetRef := DatasetRef{
		Name:   "datasetname",
		PeerID: "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
		Path:   "/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	fullDatasetRef := DatasetRef{
		Peername: "peername",
		Name:     "datasetname",
		Path:     "/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	fullIPFSDatasetRef := DatasetRef{
		Peername: "peername",
		Name:     "datasetname",
		Path:     "/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	pathOnlyDatasetRef := DatasetRef{
		Path: "/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	ipfsOnlyDatasetRef := DatasetRef{
		Path: "/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	cases := []struct {
		input  string
		expect DatasetRef
		err    string
	}{
		{"", DatasetRef{}, "cannot parse empty string as dataset reference"},
		{"peername/", peernameDatasetRef, ""},
		{"peername", peernameDatasetRef, ""},

		{"QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/", peerIDDatasetRef, ""},
		{"/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", peerIDDatasetRef, ""},

		{"peername/datasetname/", nameDatasetRef, ""},
		{"peername/datasetname", nameDatasetRef, ""},
		{"peername/datasetname/@", nameDatasetRef, ""},
		{"peername/datasetname@", nameDatasetRef, ""},

		{"/datasetname@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", idNameDatasetRef, ""},
		{"/datasetname@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/", idNameDatasetRef, ""},
		{"/datasetname/@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", idNameDatasetRef, ""},

		{"peername/datasetname/@/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", fullDatasetRef, ""},
		{"peername/datasetname@/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", fullDatasetRef, ""},

		{"/datasetname@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", idFullDatasetRef, ""},
		{"/datasetname@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", idFullDatasetRef, ""}, // 15
		{"/datasetname/@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", idFullIPFSDatasetRef, ""},
		{"/datasetname@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", idFullDatasetRef, ""},

		{"@/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", pathOnlyDatasetRef, ""},
		{"@/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", ipfsOnlyDatasetRef, ""},

		{"peername/datasetname/@/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/junk/junk/...", fullDatasetRef, ""},
		{"peername/datasetname/@/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/junk/junk/...", fullIPFSDatasetRef, ""},
		// TODO - restore. These have been removed b/c I didn't have time to make dem work properly - @b5
		// {"peername/datasetname@/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/junk/junk/...", fullIPFSDatasetRef, ""},
		// {"peername/datasetname@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/junk/junk/...", fullIPFSDatasetRef, ""},
		// {"@/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/junk/junk/...", pathOnlyDatasetRef, ""},
		// {"@network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/junk/junk/...", pathOnlyDatasetRef, ""},
		// {"@/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/junk/junk/...", ipfsOnlyDatasetRef, ""},
		// {"@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D96w1L5qAhUM5Y/junk/junk/...", ipfsOnlyDatasetRef, ""},

		// {"peername/datasetname@network/bad_hash", DatasetRef{}, "invalid PeerID: 'network'"},
		// {"peername/datasetname@bad_hash/junk/junk..", DatasetRef{}, "invalid PeerID: 'bad_hash'"},
		// {"peername/datasetname@bad_hash", DatasetRef{}, "invalid PeerID: 'bad_hash'"},

		// {"@///*(*)/", DatasetRef{}, "malformed DatasetRef string: @///*(*)/"},
		// {"///*(*)/", DatasetRef{}, "malformed DatasetRef string: ///*(*)/"},
		// {"@", DatasetRef{}, ""},
		// {"///@////", DatasetRef{}, ""},
	}

	for i, c := range cases {
		got, err := ParseDatasetRef(c.input)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		if err := CompareDatasetRef(got, c.expect); err != nil {
			t.Errorf("case %d: %s", i, err.Error())
		}
	}
}

func TestMatch(t *testing.T) {
	cases := []struct {
		a, b  string
		match bool
	}{
		{"a/b@/b/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "a/b@/b/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", true},
		{"a/b@/b/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "a/b@/b/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", true},
		{"QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/b@/b/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/b@/b/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", true},

		{"a/different_name@/b/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", "a/b@/b/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", true},
		{"different_peername/b@/b/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", "a/b@/b/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", true},
		{"different_peername/b@/b/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/b@/b/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", true},
	}

	for i, c := range cases {
		a, err := ParseDatasetRef(c.a)
		if err != nil {
			t.Errorf("case %d error parsing dataset ref a: %s", i, err.Error())
			continue
		}
		b, err := ParseDatasetRef(c.b)
		if err != nil {
			t.Errorf("case %d error parsing dataset ref b: %s", i, err.Error())
			continue
		}

		gotA := a.Match(b)
		if gotA != c.match {
			t.Errorf("case %d a.Match", i)
			continue
		}

		gotB := b.Match(a)
		if gotB != c.match {
			t.Errorf("case %d b.Match", i)
			continue
		}
	}
}

func TestEqual(t *testing.T) {
	cases := []struct {
		a, b  string
		equal bool
	}{
		{"a/b@/b/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "a/b@/b/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", true},
		{"a/b@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "a/b@/ipfs/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", false},

		{"QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1/b@/b/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1/b@/b/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", true},
		{"QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1/b@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1/b@/ipfs/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", false},

		{"a/different_name@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "a/b@/ipfs/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", false},
		{"different_peername/b@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "a/b@/ipfs/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", false},

		{"QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL/different_name@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL/b@/ipfs/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", false},
		{"QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL/b@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "a/b@/ipfs/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", false},
		{"QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL/b@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1/b@/ipfs/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL", false},
	}

	for i, c := range cases {
		a, err := ParseDatasetRef(c.a)
		if err != nil {
			t.Errorf("case %d error parsing dataset ref a: %s", i, err.Error())
			continue
		}
		b, err := ParseDatasetRef(c.b)
		if err != nil {
			t.Errorf("case %d error parsing dataset ref b: %s", i, err.Error())
			continue
		}

		gotA := a.Equal(b)
		if gotA != c.equal {
			t.Errorf("case %d a.Equal", i)
			continue
		}

		gotB := b.Equal(a)
		if gotB != c.equal {
			t.Errorf("case %d b.Equal", i)
			continue
		}
	}
}

func TestIsEmpty(t *testing.T) {
	cases := []struct {
		ref   DatasetRef
		empty bool
	}{
		{DatasetRef{}, true},
		{DatasetRef{Peername: "a"}, false},
		{DatasetRef{Name: "a"}, false},
		{DatasetRef{Path: "a"}, false},
		{DatasetRef{PeerID: "a"}, false},
	}

	for i, c := range cases {
		got := c.ref.IsEmpty()
		if got != c.empty {
			t.Errorf("case %d: %s", i, c.ref)
			continue
		}
	}
}

func TestCompareDatasetRefs(t *testing.T) {
	cases := []struct {
		a, b DatasetRef
		err  string
	}{
		{DatasetRef{}, DatasetRef{}, ""},
		{DatasetRef{Name: "a"}, DatasetRef{}, "name mismatch. a != "},
		{DatasetRef{Peername: "a"}, DatasetRef{}, "peername mismatch. a != "},
		{DatasetRef{Path: "a"}, DatasetRef{}, "path mismatch. a != "},
		{DatasetRef{PeerID: "QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1"}, DatasetRef{}, "peerID mismatch. QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1 != "},
	}

	for i, c := range cases {
		err := CompareDatasetRef(c.a, c.b)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mistmatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}
	}
}

func TestCanonicalizeDatasetRef(t *testing.T) {
	repo, err := NewMemRepo(&profile.Profile{Peername: "lucille"}, cafs.NewMapstore(), MemProfiles{})
	if err != nil {
		t.Errorf("error allocating mem repo: %s", err.Error())
		return
	}

	cases := []struct {
		input  string
		expect string
		err    string
	}{
		{"me/foo", "lucille/foo", ""},
		{"you/foo", "you/foo", ""},
		{"me/ball@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "lucille/ball@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", ""},
		// TODO - add tests that show path fulfillment
		// {"@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "lucille/ball@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", ""},
	}

	for i, c := range cases {
		ref, err := ParseDatasetRef(c.input)
		if err != nil {
			t.Errorf("case %d unexpected dataset ref parse error: %s", i, err.Error())
			continue
		}
		got := &ref

		err = CanonicalizeDatasetRef(repo, got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		if got.String() != c.expect {
			t.Errorf("case %d expected: %s, got: %s", i, c.expect, got)
			continue
		}
	}
}

func TestCanonicalizePeer(t *testing.T) {
	repo, err := NewMemRepo(&profile.Profile{Peername: "lucille", ID: "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"}, cafs.NewMapstore(), MemProfiles{})
	if err != nil {
		t.Errorf("error allocating mem repo: %s", err.Error())
		return
	}

	lucille := DatasetRef{
		PeerID:   "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
		Peername: "lucille",
	}

	ball := DatasetRef{
		PeerID:   "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
		Peername: "lucille",
		Name:     "ball",
		Path:     "/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1",
	}

	ballPeer := DatasetRef{
		PeerID:   "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
		Peername: "lucille",
		Name:     "ball",
	}

	badPeerName := DatasetRef{
		PeerID:   "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
		Peername: "bad_name",
	}

	badPeerIDGoodName := DatasetRef{
		PeerID:   "badID",
		Peername: "me",
	}

	cases := []struct {
		input           string
		inputDatasetRef DatasetRef
		expect          DatasetRef
		err             string
	}{
		{"me", DatasetRef{}, lucille, ""},
		{"lucille", DatasetRef{}, lucille, ""},
		{"QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", DatasetRef{}, lucille, ""},
		{"me/ball@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", DatasetRef{}, ball, ""},
		{"/ball@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", DatasetRef{}, ball, ""},
		{"/ball@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", DatasetRef{}, ballPeer, ""},
		{"me/ball", DatasetRef{}, ballPeer, ""},
		{"", badPeerIDGoodName, lucille, ""},
		{"", badPeerName, DatasetRef{}, "Peername and PeerID combination not valid: PeerID = QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y, Peername = lucille, but was given Peername = bad_name"},
		{"", DatasetRef{}, DatasetRef{}, ""},
		// TODO - test CanonicalizePeer works with canonicalizing peer's datasetRefs as well
	}

	for i, c := range cases {
		var (
			ref DatasetRef
			err error
		)
		if c.input != "" {
			ref, err = ParseDatasetRef(c.input)
			if err != nil {
				t.Errorf("case %d unexpected dataset ref parse error: %s", i, err.Error())
				continue
			}
		} else {
			ref = c.inputDatasetRef
		}
		got := &ref

		err = CanonicalizePeer(repo, got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		if c.err == "" {
			if got.PeerID != c.expect.PeerID {
				t.Errorf("case %d PeerID mismatch. expected: '%s', got: '%s'", i, c.expect.PeerID, got.PeerID)
			}
			if got.Peername != c.expect.Peername {
				t.Errorf("case %d Peername mismatch. expected: '%s', got: '%s'", i, c.expect.Peername, got.Peername)
			}
			if got.Name != c.expect.Name {
				t.Errorf("case %d Name mismatch. expected: '%s', got: '%s'", i, c.expect.Name, got.Name)
			}
			if got.Path != c.expect.Path {
				t.Errorf("case %d Path mismatch. expected: '%s', got: '%s'", i, c.expect.Path, got.Path)
			}
		}
	}
}
