package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/lib"
)

func TestParseGetParams(t *testing.T) {
	cases := []struct {
		description string
		url         string
		expectArgs  *lib.GetParams
		muxVars     map[string]string
	}{
		{
			"basic get",
			"/get/peer/my_ds",
			&lib.GetParams{
				Refstr: "peer/my_ds",
				Format: "json",
				All:    true,
			},
			map[string]string{"peername": "peer", "name": "my_ds"},
		},
		{
			"meta component",
			"/get/peer/my_ds/meta",
			&lib.GetParams{
				Refstr:   "peer/my_ds",
				Format:   "json",
				Selector: "meta",
				All:      true,
			},
			map[string]string{"peername": "peer", "name": "my_ds", "selector": "meta"},
		},
		{
			"body component",
			"/get/peer/my_ds/body",
			&lib.GetParams{
				Refstr:   "peer/my_ds",
				Format:   "json",
				Selector: "body",
				All:      true,
			},
			map[string]string{"peername": "peer", "name": "my_ds", "selector": "body"},
		},
		{
			"body.csv path suffix",
			"/get/peer/my_ds/body.csv",
			&lib.GetParams{
				Refstr:   "peer/my_ds",
				Format:   "csv",
				Selector: "body",
				All:      true,
			},
			map[string]string{"peername": "peer", "name": "my_ds", "selector": "body.csv"},
		},
		{
			"download body as csv",
			"/get/peer/my_ds/body?format=csv",
			&lib.GetParams{
				Refstr:   "peer/my_ds",
				Format:   "csv",
				Selector: "body",
				All:      true,
			},
			map[string]string{"peername": "peer", "name": "my_ds", "selector": "body"},
		},
		{
			"zip format",
			"/get/peer/my_ds?format=zip",
			&lib.GetParams{
				Refstr: "peer/my_ds",
				Format: "zip",
				All:    true,
			},
			map[string]string{"peername": "peer", "name": "my_ds"},
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			r, _ := http.NewRequest("GET", c.url, nil)
			if c.muxVars != nil {
				r = mux.SetURLVars(r, c.muxVars)
			}
			setRefStringFromMuxVars(r)
			args := &lib.GetParams{}
			err := lib.UnmarshalParams(r, args)
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
		muxVars     map[string]string
	}{
		{
			"get me",
			"/get/me/my_ds",
			`username "me" not allowed`,
			map[string]string{"peername": "me", "name": "my_ds"},
		},
		{
			"bad parse",
			"/get/peer/my+ds",
			`unexpected character at position 7: '+'`,
			map[string]string{"peername": "peer", "name": "my+ds"},
		},
	}
	for i, c := range badCases {
		t.Run(c.description, func(t *testing.T) {
			r, _ := http.NewRequest("GET", c.url, nil)
			if c.muxVars != nil {
				r = mux.SetURLVars(r, c.muxVars)
			}
			setRefStringFromMuxVars(r)
			args := &lib.GetParams{}
			err := lib.UnmarshalParams(r, args)
			if err == nil {
				t.Errorf("case %d: expected error, but did not get one", i)
				return
			}
			if diff := cmp.Diff(c.expectErr, err.Error()); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseGetParamsAcceptHeader(t *testing.T) {
	// Construct a request with "Accept: text/csv"
	r, _ := http.NewRequest("GET", "/get/peer/my_ds", nil)
	r.Header.Add("Accept", "text/csv")
	r = mux.SetURLVars(r, map[string]string{"peername": "peer", "name": "my_ds"})
	setRefStringFromMuxVars(r)
	args := &lib.GetParams{}
	err := lib.UnmarshalParams(r, args)
	if err != nil {
		t.Fatal(err)
	}
	expectArgs := &lib.GetParams{
		Refstr:   "peer/my_ds",
		Selector: "body",
		Format:   "csv",
		All:      true,
	}

	if diff := cmp.Diff(expectArgs, args); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Construct a request with format=csv and "Accept: text/csv", which is ok
	r, _ = http.NewRequest("GET", "/get/peer/my_ds?format=csv", nil)
	r.Header.Add("Accept", "text/csv")
	r = mux.SetURLVars(r, map[string]string{"peername": "peer", "name": "my_ds"})
	setRefStringFromMuxVars(r)
	args = &lib.GetParams{}
	err = lib.UnmarshalParams(r, args)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectArgs, args); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Construct a request with format=json and "Accept: text/csv", which is an error
	r, _ = http.NewRequest("GET", "/get/peer/my_ds?format=json", nil)
	r.Header.Add("Accept", "text/csv")
	r = mux.SetURLVars(r, map[string]string{"peername": "peer", "name": "my_ds"})
	setRefStringFromMuxVars(r)
	args = &lib.GetParams{}
	err = lib.UnmarshalParams(r, args)
	if err == nil {
		t.Error("expected to get an error, but did not get one")
	}
	expectErr := `format "json" conflicts with header "Accept: text/csv"`
	if expectErr != err.Error() {
		t.Errorf("error mismatch, expect: %q, got %q", expectErr, err)
	}
}

func TestGetZip(t *testing.T) {
	run := NewAPITestRunner(t)
	defer run.Delete()

	// Save a version of the dataset
	ds := run.BuildDataset("test_ds")
	ds.Meta = &dataset.Meta{Title: "some title"}
	ds.Readme = &dataset.Readme{ScriptBytes: []byte("# hi\n\nthis is a readme")}
	run.SaveDataset(ds, "testdata/cities/data.csv")

	// Get a zip file binary over the API
	dsHandler := NewDatasetHandlers(run.Inst, false)
	gotStatusCode, gotBodyString := APICall("/get/peer/test_ds?format=zip", dsHandler.GetHandler(""), map[string]string{"peername": "peer", "name": "test_ds"})
	if gotStatusCode != 200 {
		t.Fatalf("expected status code 200, got %d", gotStatusCode)
	}

	// Compare the API response to the expected zip file
	expectBytes, err := ioutil.ReadFile("testdata/cities/exported.zip")
	if err != nil {
		t.Fatalf("error reading expected bytes: %s", err)
	}
	if diff := cmp.Diff(string(expectBytes), gotBodyString); diff != "" {
		t.Errorf("byte mismatch (-want +got):\n%s", diff)
	}
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

	actualStatusCode, actualBody := APICall("/get/peer/test_ds", dsHandler.GetHandler(""), map[string]string{"peername": "peer", "name": "test_ds"})
	assertStatusCode(t, "get dataset", actualStatusCode, 200)
	got := datasetJSONResponse(t, actualBody)
	dstest.CompareGoldenDatasetAndUpdateIfEnvVarSet(t, "testdata/expect/TestDatasetGet.test_ds.json", got)

	// Get csv body using "body.csv" suffix
	actualStatusCode, actualBody = APICall("/get/peer/test_ds/body.csv", dsHandler.GetHandler(""), map[string]string{"peername": "peer", "name": "test_ds", "selector": "body.csv"})
	expectBody := "city,pop,avg_age,in_usa\ntoronto,40000000,55.5,false\nnew york,8500000,44.4,true\nchicago,300000,44.4,true\nchatham,35000,65.25,true\nraleigh,250000,50.65,true\n"
	assertStatusCode(t, "get body.csv using suffix", actualStatusCode, 200)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Can get zip file
	actualStatusCode, _ = APICall("/get/peer/test_ds?format=zip", dsHandler.GetHandler(""), map[string]string{"peername": "peer", "name": "test_ds"})
	assertStatusCode(t, "get zip file", actualStatusCode, 200)

	// Can get a single component
	actualStatusCode, _ = APICall("/get/peer/test_ds/meta", dsHandler.GetHandler(""), map[string]string{"peername": "peer", "name": "test_ds", "selector": "meta"})
	assertStatusCode(t, "get meta component", actualStatusCode, 200)

	// Can get at an ipfs version
	actualStatusCode, _ = APICall("/get/peer/test_ds/at/mem/QmeTvt83npHg4HoxL8bp8yz5bmG88hUVvRc5k9taW8uxTr", dsHandler.GetHandler(""), map[string]string{"peername": "peer", "name": "test_ds", "fs": "mem", "hash": "QmeTvt83npHg4HoxL8bp8yz5bmG88hUVvRc5k9taW8uxTr"})
	assertStatusCode(t, "get at content-addressed version", actualStatusCode, 200)

	// Error 404 if ipfs version doesn't exist
	actualStatusCode, _ = APICall("/get/peer/test_ds/at/mem/QmissingEJUqFWNfdiPTPtxyba6wf86TmbQe1nifpZCRH6", dsHandler.GetHandler(""), map[string]string{"peername": "peer", "name": "test_ds", "fs": "mem", "hash": "QmissingEJUqFWNfdiPTPtxyba6wf86TmbQe1nifpZCRH6"})
	assertStatusCode(t, "get missing content-addressed version", actualStatusCode, 404)

	// Error 400 due to unknown component
	actualStatusCode, _ = APICall("/get/peer/test_ds/dunno", dsHandler.GetHandler(""), map[string]string{"peername": "peer", "name": "test_ds", "selector": "dunno"})
	assertStatusCode(t, "unknown component", actualStatusCode, 400)

	// Error 400 due to parse error of dsref
	actualStatusCode, _ = APICall("/get/peer/test+ds", dsHandler.GetHandler(""), map[string]string{"peername": "peer", "name": "test+ds"})
	assertStatusCode(t, "invalid dsref", actualStatusCode, 400)
}

func assertStatusCode(t *testing.T, description string, actualStatusCode, expectStatusCode int) {
	t.Helper()
	if expectStatusCode != actualStatusCode {
		t.Errorf("%s: expected status code %d, got %d", description, expectStatusCode, actualStatusCode)
	}
}

func datasetJSONResponse(t *testing.T, body string) *dataset.Dataset {
	t.Helper()
	res := struct {
		Data struct {
			Dataset *dataset.Dataset
		}
	}{}
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		t.Fatal(err)
	}
	return res.Data.Dataset
}
