package repo

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

var cases = []struct {
	ref         reporef.DatasetRef
	String      string
	Absolute    string
	AliasString string
}{
	{reporef.DatasetRef{
		Peername: "peername",
	}, "peername", "peername", "peername"},
	{reporef.DatasetRef{
		Peername: "peername",
		Name:     "datasetname",
	}, "peername/datasetname", "peername/datasetname", "peername/datasetname"},

	{reporef.DatasetRef{
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
	}, "", "@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", ""},
	{reporef.DatasetRef{
		Path: "/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1",
	}, "@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", ""},

	{reporef.DatasetRef{
		Peername:  "peername",
		Name:      "datasetname",
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
	}, "peername/datasetname", "peername/datasetname@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", "peername/datasetname"},
	{reporef.DatasetRef{
		Peername:  "peername",
		Name:      "datasetname",
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
		Path:      "/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1",
	}, "peername/datasetname@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "peername/datasetname@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "peername/datasetname"},

	{reporef.DatasetRef{
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
		Peername:  "lucille",
	}, "lucille", "lucille@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", "lucille"},
	{reporef.DatasetRef{
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
		Peername:  "lucille",
		Name:      "ball",
		Path:      "/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1",
	}, "lucille/ball@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "lucille/ball@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "lucille/ball"},

	{reporef.DatasetRef{
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
		Peername:  "bad_name",
	}, "bad_name", "bad_name@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", "bad_name"},
	// TODO - this used to be me@badId, which isn't very useful, but at least provided coding parity
	// might be worth revisiting
	{reporef.DatasetRef{
		ProfileID: profile.IDRawByteString("badID"),
		Peername:  "me",
	}, "me", "me@C6mUq3y", "me"},
}

func TestDatasetString(t *testing.T) {
	for i, c := range cases {
		if c.ref.String() != c.String {
			t.Errorf("case %d:\n%s\n%s", i, c.ref.String(), c.String)
			continue
		}
	}
}

func TestDatasetAbsolute(t *testing.T) {
	for i, c := range cases {
		if c.ref.Absolute() != c.Absolute {
			t.Errorf("case %d:\n%s\n%s", i, c.ref.Absolute(), c.Absolute)
			continue
		}
	}
}

func TestDatasetAliasString(t *testing.T) {
	for i, c := range cases {
		if c.ref.AliasString() != c.AliasString {
			t.Errorf("case %d:\n%s\n%s", i, c.ref.AliasString(), c.AliasString)
			continue
		}
	}
}

func TestParseDatasetRef(t *testing.T) {
	peernameDatasetRef := reporef.DatasetRef{
		Peername: "peername",
	}

	nameDatasetRef := reporef.DatasetRef{
		Peername: "peername",
		Name:     "datasetname",
	}

	peerIDDatasetRef := reporef.DatasetRef{
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
	}

	idNameDatasetRef := reporef.DatasetRef{
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
		Name:      "datasetname",
	}

	idFullDatasetRef := reporef.DatasetRef{
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
		Name:      "datasetname",
		Path:      "/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	idFullIPFSDatasetRef := reporef.DatasetRef{
		Name:      "datasetname",
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
		Path:      "/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	fullDatasetRef := reporef.DatasetRef{
		Peername: "peername",
		Name:     "datasetname",
		Path:     "/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	fullIPFSDatasetRef := reporef.DatasetRef{
		Peername: "peername",
		Name:     "datasetname",
		Path:     "/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	pathOnlyDatasetRef := reporef.DatasetRef{
		Path: "/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	ipfsOnlyDatasetRef := reporef.DatasetRef{
		Path: "/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
	}

	mapDatasetRef := reporef.DatasetRef{
		Path: "/map/QmcQsi93yUryyWvw6mPyDNoKRb7FcBx8QGBAeJ25kXQjnC",
	}

	cases := []struct {
		input  string
		expect reporef.DatasetRef
		err    string
	}{
		{"", reporef.DatasetRef{}, "repo: empty dataset reference"},
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
		{"@/map/QmcQsi93yUryyWvw6mPyDNoKRb7FcBx8QGBAeJ25kXQjnC", mapDatasetRef, ""},

		{"peername/datasetname/@/network/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/junk/junk/...", fullDatasetRef, ""},
		{"peername/datasetname/@/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/junk/junk/...", fullIPFSDatasetRef, ""},
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
		ref   reporef.DatasetRef
		empty bool
	}{
		{reporef.DatasetRef{}, true},
		{reporef.DatasetRef{Peername: "a"}, false},
		{reporef.DatasetRef{Name: "a"}, false},
		{reporef.DatasetRef{Path: "a"}, false},
		{reporef.DatasetRef{ProfileID: profile.IDRawByteString("a")}, false},
	}

	for i, c := range cases {
		got := c.ref.IsEmpty()
		if got != c.empty {
			t.Errorf("case %d: %s", i, c.ref)
			continue
		}
	}
}

func TestCompareDatasets(t *testing.T) {
	cases := []struct {
		a, b reporef.DatasetRef
		err  string
	}{
		{reporef.DatasetRef{}, reporef.DatasetRef{}, ""},
		{reporef.DatasetRef{Name: "a"}, reporef.DatasetRef{}, "Name mismatch. a != "},
		{reporef.DatasetRef{Peername: "a"}, reporef.DatasetRef{}, "Peername mismatch. a != "},
		{reporef.DatasetRef{Path: "a"}, reporef.DatasetRef{}, "Path mismatch. a != "},
		{reporef.DatasetRef{ProfileID: profile.IDB58MustDecode("QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1")}, reporef.DatasetRef{}, "PeerID mismatch. QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1 != "},
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
	ctx := context.Background()
	lucille := &profile.Profile{ID: profile.IDRawByteString("a"), Peername: "lucille", PrivKey: privKey}
	carla := &profile.Profile{ID: profile.IDRawByteString("b"), Peername: "carla"}

	fs, err := muxfs.New(ctx, []qfs.Config{
		{Type: "mem"},
	})

	if err != nil {
		t.Fatal(err)
	}

	memRepo, err := NewMemRepoWithProfile(ctx, lucille, fs, event.NilBus)
	if err != nil {
		t.Errorf("error allocating mem repo: %s", err.Error())
		return
	}
	rs := memRepo.MemRefstore
	for _, r := range []reporef.DatasetRef{
		{ProfileID: lucille.ID, Peername: "lucille", Name: "foo", Path: "/ipfs/QmTest"},
		{ProfileID: carla.ID, Peername: carla.Peername, Name: "hockey_stats", Path: "/ipfs/QmTest2"},
		{ProfileID: lucille.ID, Peername: "lucille", Name: "ball", Path: "/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1"},
	} {
		if err := rs.PutRef(r); err != nil {
			t.Fatal(err)
		}
	}

	// TODO - this points to a problem in our thinking. placing a reference in the refstore doesn't automatically
	// add it to the profile store. This can lead to bugs higher up the stack, but points to an architectural challenge:
	// the profile store is supposed to be the source of truth for profiles, and that isn't being enforced here.
	// Moreover, what's the correct thing to do when adding a ref to the refstore who's profile information is not in the profile
	// store, or worse, doesn't match the profile store?
	// There's an implied hierarchy of profile store > refstore that isn't being enforced in code, and should be.
	if err := memRepo.Profiles().PutProfile(ctx, carla); err != nil {
		t.Fatal(err.Error())
	}

	cases := []struct {
		input  string
		expect string
		err    string
	}{
		{"me/foo", "lucille/foo@/ipfs/QmTest", ""},
		{"carla/hockey_stats", "carla/hockey_stats@/ipfs/QmTest2", ""},
		{"lucille/ball", "lucille/ball@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", ""},
		{"me/ball@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "lucille/ball@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", ""},
		{"@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "lucille/ball@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", ""},
		{"renamed/ball@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", "lucille/ball@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", ""},
	}

	for i, c := range cases {
		ref, err := ParseDatasetRef(c.input)
		if err != nil {
			t.Errorf("case %d unexpected dataset ref parse error: %s", i, err.Error())
			continue
		}
		got := &ref

		err = canonicalizeDatasetRef(ctx, memRepo, got)
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

func TestCanonicalizeDatasetRefFSI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peer := "lucille"
	prof := &profile.Profile{ID: profile.IDRawByteString("a"), Peername: peer, PrivKey: privKey}
	fs, err := muxfs.New(ctx, []qfs.Config{
		{Type: "mem"},
	})
	if err != nil {
		t.Fatal(err)
	}

	memRepo, err := NewMemRepoWithProfile(ctx, prof, fs, event.NilBus)
	if err != nil {
		t.Fatal(err)
	}
	id := prof.ID

	rs := memRepo.MemRefstore
	rs.PutRef(reporef.DatasetRef{ProfileID: id, Peername: peer, Name: "apple", Path: "/ipfs/QmTest1"})
	rs.PutRef(reporef.DatasetRef{ProfileID: id, Peername: peer, Name: "banana", Path: "/ipfs/QmTest2", FSIPath: "/path/to/dataset"})

	goodCases := []struct {
		input      string
		expectPath string
		expectFSI  string
	}{
		{"me/apple", "/ipfs/QmTest1", ""},
		{"me/apple@/ipfs/QmTest1", "/ipfs/QmTest1", ""},
		{"me/apple@/ipfs/QmTest1Prev", "/ipfs/QmTest1Prev", ""},
		{"me/banana", "/ipfs/QmTest2", "/path/to/dataset"},
		{"me/banana@/ipfs/QmTest2", "/ipfs/QmTest2", "/path/to/dataset"},
		{"me/banana@/ipfs/QmTest2Prev", "/ipfs/QmTest2Prev", "/path/to/dataset"},
	}

	for i, c := range goodCases {
		ref, err := ParseDatasetRef(c.input)
		if err != nil {
			t.Errorf("case %d unexpected dataset ref parse error: %s", i, err)
			continue
		}
		got := &ref

		err = canonicalizeDatasetRef(ctx, memRepo, got)
		if err != nil {
			t.Errorf("case %d got error: %s", i, err)
			continue
		}

		if got.Path != c.expectPath {
			t.Errorf("case %d expected path: %s, got: %s", i, c.expectPath, got.Path)
			continue
		}
		if got.FSIPath != c.expectFSI {
			t.Errorf("case %d expected FSI path: %s, got: %s", i, c.expectFSI, got.FSIPath)
			continue
		}
	}
}

func TestCanonicalizeProfile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prof := &profile.Profile{Peername: "lucille", ID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"), PrivKey: privKey}
	fs, err := muxfs.New(ctx, []qfs.Config{
		{Type: "mem"},
	})
	if err != nil {
		t.Fatal(err)
	}

	repo, err := NewMemRepoWithProfile(ctx, prof, fs, event.NilBus)
	if err != nil {
		t.Errorf("error allocating mem repo: %s", err.Error())
		return
	}

	lucille := reporef.DatasetRef{
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
		Peername:  "lucille",
	}

	ball := reporef.DatasetRef{
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
		Peername:  "lucille",
		Name:      "ball",
		Path:      "/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1",
	}

	ballPeer := reporef.DatasetRef{
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
		Peername:  "lucille",
		Name:      "ball",
	}

	renamePeerName := reporef.DatasetRef{
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
		Peername:  "lucy",
		Name:      "ball",
		Path:      "/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1",
	}

	badProfileIDGoodName := reporef.DatasetRef{
		ProfileID: profile.IDRawByteString("badID"),
		Peername:  "me",
	}

	cases := []struct {
		input        string
		inputDataset reporef.DatasetRef
		expect       reporef.DatasetRef
		err          string
	}{
		{"me", reporef.DatasetRef{}, lucille, ""},
		{"lucille", reporef.DatasetRef{}, lucille, ""},
		{"QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", reporef.DatasetRef{}, lucille, ""},
		{"me/ball@/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", reporef.DatasetRef{}, ball, ""},
		{"/ball@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", reporef.DatasetRef{}, ball, ""},
		{"/ball@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", reporef.DatasetRef{}, ballPeer, ""},
		{"me/ball", reporef.DatasetRef{}, ballPeer, ""},
		{"", badProfileIDGoodName, lucille, ""},
		{"/ball@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1", renamePeerName, ball, ""},
		{"", reporef.DatasetRef{}, reporef.DatasetRef{}, ""},
	}

	for i, c := range cases {
		var (
			ref reporef.DatasetRef
			err error
		)
		if c.input != "" {
			ref, err = ParseDatasetRef(c.input)
			if err != nil {
				t.Errorf("case %d unexpected dataset ref parse error: %s", i, err.Error())
				continue
			}
		} else {
			ref = c.inputDataset
		}
		got := &ref

		err = canonicalizeProfile(ctx, repo, got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		if c.err == "" {
			if got.ProfileID != c.expect.ProfileID {
				t.Errorf("case %d ProfileID mismatch. expected: '%s', got: '%s'", i, c.expect.ProfileID, got.ProfileID)
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

func TestConvertToDsref(t *testing.T) {
	ref := reporef.DatasetRef{
		ProfileID: profile.IDB58MustDecode("QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"),
		Peername:  "lucy",
		Name:      "ball",
		Path:      "/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1",
	}
	expect := dsref.Ref{
		Username:  "lucy",
		Name:      "ball",
		ProfileID: "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y",
		Path:      "/ipfs/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1",
	}
	got := reporef.ConvertToDsref(ref)

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
