package repo

import (
	"github.com/ipfs/go-datastore"
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
		{"peer_name/dataset_name@/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", &DatasetRef{Peername: "peer_name", Name: "dataset_name", Path: datastore.NewKey("/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y")}, ""},
		{"peer_name/dataset_name@QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", &DatasetRef{Peername: "peer_name", Name: "dataset_name", Path: datastore.NewKey("/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y")}, ""},
		{"dataset_name", &DatasetRef{Name: "dataset_name"}, ""},
		// {"/not_ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", &DatasetRef{}, ""},
		{"dataset_name", &DatasetRef{Name: "dataset_name"}, ""},
		{"QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y", &DatasetRef{Path: datastore.NewKey("/ipfs/QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y")}, ""},
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
