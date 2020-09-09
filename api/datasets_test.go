package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
)

func TestParseGetReqArgs(t *testing.T) {
	cases := []struct {
		description string
		url         string
		expectArgs  *GetReqArgs
	}{
		{
			"basic get",
			"/get/peer/my_ds",
			&GetReqArgs{
				Ref: dsref.MustParse("peer/my_ds"),
				Params: lib.GetParams{
					Refstr: "peer/my_ds",
					Format: "json",
					Limit:  100,
				},
			},
		},
		{
			"meta component",
			"/get/peer/my_ds?component=meta",
			&GetReqArgs{
				Ref: dsref.MustParse("peer/my_ds"),
				Params: lib.GetParams{
					Refstr:   "peer/my_ds",
					Format:   "json",
					Selector: "meta",
					Limit:    100,
				},
			},
		},
		{
			"body component",
			"/get/peer/my_ds?component=body",
			&GetReqArgs{
				Ref: dsref.MustParse("peer/my_ds"),
				Params: lib.GetParams{
					Refstr:   "peer/my_ds",
					Format:   "json",
					Selector: "body",
					Limit:    100,
				},
			},
		},
		{
			"body.csv path suffix",
			"/get/peer/my_ds/body.csv",
			&GetReqArgs{
				Ref:         dsref.MustParse("peer/my_ds"),
				RawDownload: true,
				Params: lib.GetParams{
					Refstr:   "peer/my_ds",
					Format:   "csv",
					Selector: "body",
					Limit:    100,
				},
			},
		},
		{
			"download body as csv",
			"/get/peer/my_ds?download=true&format=csv&component=body",
			&GetReqArgs{
				Ref:         dsref.MustParse("peer/my_ds"),
				RawDownload: true,
				Params: lib.GetParams{
					Refstr:   "peer/my_ds",
					Format:   "csv",
					Selector: "body",
					Limit:    100,
				},
			},
		},
		{
			"zip format",
			"/get/peer/my_ds?format=zip",
			&GetReqArgs{
				Ref: dsref.MustParse("peer/my_ds"),
				Params: lib.GetParams{
					Refstr: "peer/my_ds",
					Format: "zip",
					Limit:  100,
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			r, _ := http.NewRequest("GET", c.url, nil)
			reqPath := trimGetOrBodyPrefix(r.URL.Path)
			args, err := parseGetReqArgs(r, reqPath)
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(c.expectArgs, args); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}

	badCases := []struct {
		description string
		url         string
		expectErr   string
	}{
		{
			"get me",
			"/get/me/my_ds",
			`username "me" not allowed`,
		},
		{
			"bad parse",
			"/get/peer/my+ds",
			`unexpected character at position 7: '+'`,
		},
		{
			"invalid format",
			"/get/peer/my_ds?format=csv",
			`only supported formats are "json" and "zip", unless using download parameter or Accept header is set to "text/csv"`,
		},
	}
	for _, c := range badCases {
		t.Run(c.description, func(t *testing.T) {
			r, _ := http.NewRequest("GET", c.url, nil)
			reqPath := trimGetOrBodyPrefix(r.URL.Path)
			_, err := parseGetReqArgs(r, reqPath)
			if err == nil {
				t.Errorf("expected error, but did not get one")
				return
			}
			if diff := cmp.Diff(c.expectErr, err.Error()); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func trimGetOrBodyPrefix(text string) string {
	if strings.HasPrefix(text, "/get/") {
		text = strings.TrimPrefix(text, "/get/")
	}
	if strings.HasPrefix(text, "/body/") {
		text = strings.TrimPrefix(text, "/body/")
	}
	return text
}

func TestDatasetGet(t *testing.T) {
	run := NewAPITestRunner(t)
	defer run.Delete()

	dsHandler := NewDatasetHandlers(run.Inst, false)

	ds := dataset.Dataset{
		Name: "test_ds",
		Meta: &dataset.Meta{
			Title: "title one",
		},
	}
	run.SaveDataset(&ds, "testdata/cities/data.csv")

	// TODO(dustmop): Would be nice to have a "fuzzy" json comparison, either as a third-party
	// library, or something we develop, that would make tests like this easier to read and
	// reason about. There's certain values in this json that we really care about (format),
	// and then there's some we don't care about at all (signature).

	actualStatusCode, actualBody := APICall("/get/peer/test_ds", dsHandler.GetHandler)
	expectBody := `{"data":{"peername":"peer","name":"test_ds","path":"/map/QmRvuuXS4cPeZnxMMaXKaqMjEa3tunqZd6r3c4HEH2h3KN","dataset":{"bodyPath":"/map/QmVYgdpvgnq3FABZFVWUgxr7UCwNSRJz97vBU9YX5g5pQ4","commit":{"author":{"id":"QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"},"message":"created dataset from data.csv","path":"/map/QmbsySjGEJUqFWNfdiPTPtxyba6wf86TmbQe1nifpZCRH6","qri":"cm:0","signature":"TzHbqw7oRcqoGYhTraiJ9fSGzUUoHA653jNYGsBgbkUbBUkTh/ymTHZSfhwIKQlAqfwiOqB2rbmA4hu2MAYVxNRLfPlUYAr38juyTosI6cljYdzLfNk2L788iFSQcYnJ9CgiHYQlhmpUjh3irFY1nDIuaiPL1vDxH5yGsuI3qiD9DVvu2P6f6GSGMkQzpSv3jDWQbqo5LyyU7gaJBIaJW6Q9vKSB/mRRjDfHtOP2pMH/pf17q35+HaGHd+XEg/6a6X3IWwjsRros029lH6SDCbUaPXB6H3Cy5gRLoZp7K3mU026JucogrVHqRsZmVCx+vaVJ/MCpQhfYg6F8m8z2fA==","timestamp":"2001-01-01T01:01:01.000000001Z","title":"created dataset from data.csv"},"meta":{"qri":"md:0","title":"title one"},"name":"test_ds","path":"/map/QmRvuuXS4cPeZnxMMaXKaqMjEa3tunqZd6r3c4HEH2h3KN","peername":"peer","qri":"ds:0","structure":{"checksum":"QmVYgdpvgnq3FABZFVWUgxr7UCwNSRJz97vBU9YX5g5pQ4","depth":2,"entries":5,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":154,"qri":"st:0","schema":{"items":{"items":[{"title":"city","type":"string"},{"title":"pop","type":"integer"},{"title":"avg_age","type":"number"},{"title":"in_usa","type":"boolean"}],"type":"array"},"type":"array"}}},"published":false},"meta":{"code":200}}`
	assertStatusCode(t, "get dataset", actualStatusCode, 200)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Get csv body using "body.csv" suffix
	actualStatusCode, actualBody = APICall("/get/peer/test_ds/body.csv", dsHandler.GetHandler)
	expectBody = "city,pop,avg_age,in_usa\ntoronto,40000000,55.5,false\nnew york,8500000,44.4,true\nchicago,300000,44.4,true\nchatham,35000,65.25,true\nraleigh,250000,50.65,true\n"
	assertStatusCode(t, "get body.csv using suffix", actualStatusCode, 200)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Same csv body, using download=true and format=csv
	actualStatusCode, actualBody = APICall("/get/peer/test_ds?download=true&format=csv", dsHandler.GetHandler)
	assertStatusCode(t, "get csv body using download=true and format=csv", actualStatusCode, 200)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Can get zip file
	actualStatusCode, _ = APICall("/get/peer/test_ds?format=zip", dsHandler.GetHandler)
	assertStatusCode(t, "get zip file", actualStatusCode, 200)

	// Can get a single component
	actualStatusCode, _ = APICall("/get/peer/test_ds?component=meta", dsHandler.GetHandler)
	assertStatusCode(t, "get meta component", actualStatusCode, 200)

	// Can get at an ipfs version
	actualStatusCode, _ = APICall("/get/peer/test_ds/at/map/QmbsySjGEJUqFWNfdiPTPtxyba6wf86TmbQe1nifpZCRH6", dsHandler.GetHandler)
	assertStatusCode(t, "get at ipfs version", actualStatusCode, 200)

	// Error 404 if ipfs version doesn't exist
	actualStatusCode, _ = APICall("/get/peer/test_ds/at/map/QmissingEJUqFWNfdiPTPtxyba6wf86TmbQe1nifpZCRH6", dsHandler.GetHandler)
	assertStatusCode(t, "get missing ipfs", actualStatusCode, 404)

	// Error 400 due to format=csv without download=true
	actualStatusCode, _ = APICall("/get/peer/test_ds?format=csv", dsHandler.GetHandler)
	assertStatusCode(t, "using format=csv", actualStatusCode, 400)

	// Error 400 due to unknown component
	actualStatusCode, _ = APICall("/get/peer/test_ds?component=dunno", dsHandler.GetHandler)
	assertStatusCode(t, "unknown component", actualStatusCode, 400)

	// Error 400 due to parse error of dsref
	actualStatusCode, _ = APICall("/get/peer/test+ds", dsHandler.GetHandler)
	assertStatusCode(t, "invalid dsref", actualStatusCode, 400)

	// Old style /body endpoint still works
	actualStatusCode, _ = APICall("/body/peer/test_ds", dsHandler.BodyHandler)
	assertStatusCode(t, "old style /body endpoint", actualStatusCode, 200)

	// Old style /body endpoint cannot use a component
	actualStatusCode, _ = APICall("/body/peer/test_ds?component=meta", dsHandler.BodyHandler)
	assertStatusCode(t, "/body endpoint with meta component", actualStatusCode, 400)
}

func assertStatusCode(t *testing.T, description string, actualStatusCode, expectStatusCode int) {
	if expectStatusCode != actualStatusCode {
		t.Errorf("%s: expected status code %d, got %d", description, expectStatusCode, actualStatusCode)
	}
}
