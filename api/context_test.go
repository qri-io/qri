package api

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/qri-io/qri/repo"
)

func TestDatasetRefFromReq(t *testing.T) {
	cases := []struct {
		url      string
		expected repo.DatasetRef
		err      string
	}{
		{"http://localhost:2503/peername", repo.DatasetRef{Peername: "peername"}, ""},
		{"http://localhost:2503/peername?limit=10&offset=2", repo.DatasetRef{Peername: "peername"}, ""},
		{"http://localhost:2503/peername/datasetname", repo.DatasetRef{Peername: "peername", Name: "datasetname"}, ""},
		{"http://localhost:2503/peername/datasetname/at/ipfs/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD", repo.DatasetRef{Peername: "peername", Name: "datasetname", Path: "/ipfs/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD"}, ""},
		{"http://localhost:2503/peername/datasetname/at/ntwk/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD", repo.DatasetRef{Peername: "peername", Name: "datasetname", Path: "/ntwk/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD"}, ""},
		{"http://localhost:2503/peername/datasetname/at/map/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD/dataset.json", repo.DatasetRef{Peername: "peername", Name: "datasetname", Path: "/map/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD"}, ""},
		{"http://localhost:2503/peername/datasetname/at/ipfs/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD", repo.DatasetRef{Peername: "peername", Name: "datasetname", Path: "/ipfs/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD"}, ""},
		{"http://google.com:8000/peername/datasetname/at/ipfs/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD", repo.DatasetRef{Peername: "peername", Name: "datasetname", Path: "/ipfs/QmdWJ7RnFj3SdWW85mR4AYP17C8dRPD9eUPyTqUxVyGMgD"}, ""},
		{"http://google.com:8000/peername", repo.DatasetRef{Peername: "peername"}, ""},
		// {"http://google.com/peername", repo.DatasetRef{Peername: "peername"}, ""},
		{"/peername", repo.DatasetRef{Peername: "peername"}, ""},
		{"http://www.fkjhdekaldschjxilujkjkjknwjkn.org/peername/datasetname/", repo.DatasetRef{Peername: "peername", Name: "datasetname"}, ""},
		{"http://example.com", repo.DatasetRef{}, ""},
		{"", repo.DatasetRef{}, ""},
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
		if err := repo.CompareDatasetRef(got, c.expected); err != nil {
			t.Errorf("case %d: %s", i, err.Error())
		}
	}
}
