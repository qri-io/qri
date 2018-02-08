package repo

import (
	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/qri/repo/profile"
	"testing"
)

func TestParseDatasetRef(t *testing.T) {
	cases := []struct {
		input  string
		expect DatasetRef
		err    string
	}{
		{"", DatasetRef{}, "cannot parse empty string as dataset reference"},
		{"peer_name/dataset_name", DatasetRef{Peername: "peer_name", Name: "dataset_name"}, ""},
		{"peer_name/dataset_name@/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", DatasetRef{Peername: "peer_name", Name: "dataset_name", Path: "/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"}, ""},
		{"peer_name/dataset_name@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", DatasetRef{Peername: "peer_name", Name: "dataset_name", Path: "/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"}, ""},
		{"peer_name", DatasetRef{Peername: "peer_name"}, ""},
		{"tangelo_saluki/dog_names", DatasetRef{Peername: "tangelo_saluki", Name: "dog_names"}, ""},
		// {"/not_ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", DatasetRef{}, ""},
		{"QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", DatasetRef{Path: "/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"}, ""},
		{"/ipfs/Qmd3y5VuSLtEyfWk3Hud7BxgSxB6LxrBV16fcXXTPG7zDe", DatasetRef{Path: "/ipfs/Qmd3y5VuSLtEyfWk3Hud7BxgSxB6LxrBV16fcXXTPG7zDe"}, ""},
		{"peer/test@/map/QmbFrEXU5RTKcoMVoeDqpFxZyYqcYAXLoWgsYJuRtJg1Ht", DatasetRef{Peername: "peer", Name: "test", Path: "/map/QmbFrEXU5RTKcoMVoeDqpFxZyYqcYAXLoWgsYJuRtJg1Ht"}, ""},
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
		{"a/b@/b/foo", "a/b@/b/foo", true},
		{"a/b@/b/foo", "a/b@/b/bar", true},

		{"a/different_name@/b/bar", "a/b@/b/bar", true},
		{"different_peername/b@/b/bar", "a/b@/b/bar", true},
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
		{"a/b@/b/foo", "a/b@/b/foo", true},
		{"a/b@/ipfs/foo", "a/b@/ipfs/bar", false},

		{"a/different_name@/ipfs/foo", "a/b@/ipfs/bar", false},
		{"different_peername/b@/ipfs/foo", "a/b@/ipfs/bar", false},
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
	}

	for i, c := range cases {
		err := CompareDatasetRef(c.a, c.b)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mistmatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}
	}
}

func TestIsLocalRef(t *testing.T) {
	repo, err := NewMemRepo(&profile.Profile{Peername: "lucille"}, memfs.NewMapstore(), MemPeers{}, &analytics.Memstore{})
	if err != nil {
		t.Errorf("error allocating mem repo: %s", err.Error())
		return
	}

	cases := []struct {
		input  string
		expect bool
		err    string
	}{
		{"me", true, ""},
		{"you", false, ""},
		{"them", false, ""},
		{"you/foo", false, ""},
		{"me/foo", true, ""},
		{"lucille/foo", true, ""},
		// TODO - add local datasets to memrepo, have them return true
	}

	for i, c := range cases {
		ref, err := ParseDatasetRef(c.input)
		if err != nil {
			t.Errorf("case %d unexpected dataset ref parse error: %s", i, err.Error())
			continue
		}

		got, err := IsLocalRef(repo, ref)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		if got != c.expect {
			t.Errorf("case %d expected: %t", i, c.expect)
			continue
		}
	}
}
