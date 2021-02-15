package api

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/dsref"
)

func TestDatasetRefFromReq(t *testing.T) {
	cases := []struct {
		url      string
		expected dsref.Ref
		err      string
	}{
		{"http://localhost:2503/peername", dsref.Ref{Username: "peername"}, ""},
		{"http://localhost:2503/peername?limit=10&offset=2", dsref.Ref{Username: "peername"}, ""},
		{"http://localhost:2503/peername/datasetname", dsref.Ref{Username: "peername", Name: "datasetname"}, ""},
		{"http://localhost:2503/peername/datasetname/at/ipfs/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD", dsref.Ref{Username: "peername", Name: "datasetname", Path: "/ipfs/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD"}, ""},
		// {"http://localhost:2503/peername/datasetname/at/ntwk/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD", dsref.Ref{Username: "peername", Name: "datasetname", Path: "/ntwk/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD"}, ""},
		{"http://localhost:2503/peername/datasetname/at/mem/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD/dataset.json", dsref.Ref{Username: "peername", Name: "datasetname", Path: "/mem/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD"}, "unexpected character at position 72: '/'"},
		{"http://localhost:2503/peername/datasetname/at/ipfs/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD", dsref.Ref{Username: "peername", Name: "datasetname", Path: "/ipfs/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD"}, ""},
		{"http://google.com:8000/peername/datasetname/at/ipfs/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD", dsref.Ref{Username: "peername", Name: "datasetname", Path: "/ipfs/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD"}, ""},
		{"http://google.com:8000/peername", dsref.Ref{Username: "peername"}, ""},
		// {"http://google.com/peername", dsref.Ref{Username: "peername"}, ""},
		{"/peername", dsref.Ref{Username: "peername"}, ""},
		{"http://www.fkjhdekaldschjxilujkjkjknwjkn.org/peername/datasetname", dsref.Ref{Username: "peername", Name: "datasetname"}, ""},
		{"http://www.fkjhdekaldschjxilujkjkjknwjkn.org/peername/datasetname/", dsref.Ref{Username: "peername", Name: "datasetname"}, "unexpected character at position 20: '/'"},
		{"http://example.com", dsref.Ref{}, ""},
		{"", dsref.Ref{}, ""},
	}

	for i, c := range cases {
		r, err := http.NewRequest("GET", c.url, bytes.NewReader(nil))
		if err != nil {
			t.Errorf("case %d, error making request: %s", i, err)
		}
		got, err := DatasetRefFromReq(r)
		if (c.err != "" && err == nil) || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, error mismatch: expected '%s' but got '%s'", i, c.err, err)
			continue
		}
		if diff := cmp.Diff(c.expected, got); diff != "" {
			t.Errorf("case %d: output mismatch (-want +got):\n%s", i, diff)
		}
	}
}
