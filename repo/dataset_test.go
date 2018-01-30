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
		expect *DatasetRef
		err    string
	}{
		{"", nil, "cannot parse empty string as dataset reference"},
		{"peer_name/dataset_name", &DatasetRef{Peername: "peer_name", Name: "dataset_name"}, ""},
		{"peer_name/dataset_name@/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", &DatasetRef{Peername: "peer_name", Name: "dataset_name", Path: "/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"}, ""},
		{"peer_name/dataset_name@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", &DatasetRef{Peername: "peer_name", Name: "dataset_name", Path: "/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"}, ""},
		{"peer_name", &DatasetRef{Peername: "peer_name"}, ""},
		{"tangelo_saluki/dog_names", &DatasetRef{Peername: "tangelo_saluki", Name: "dog_names"}, ""},
		// {"/not_ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", &DatasetRef{}, ""},
		{"QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", &DatasetRef{Path: "/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"}, ""},
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

func TestCompareDatasetRefs(t *testing.T) {
	cases := []struct {
		a, b *DatasetRef
		err  string
	}{
		{nil, nil, ""},
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
