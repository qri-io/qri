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

func TestGetZip(t *testing.T) {
	run := NewAPITestRunner(t)
	defer run.Delete()

	// Save a version of the dataset
	ds := run.BuildDataset("test_ds")
	ds.Meta = &dataset.Meta{Title: "some title"}
	ds.Readme = &dataset.Readme{ScriptBytes: []byte("# hi\n\nthis is a readme")}
	run.SaveDataset(ds, "testdata/cities/data.csv")

	// Get a zip file binary over the API
	gotStatusCode, gotBodyString := APICall("/get/peer/test_ds?format=zip", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds"})
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

	ds := dataset.Dataset{
		Name: "test_ds",
		Meta: &dataset.Meta{
			Title: "title one",
		},
	}
	run.SaveDataset(&ds, "testdata/cities/data.csv")

	actualStatusCode, actualBody := APICall("/get/peer/test_ds", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds"})
	assertStatusCode(t, "get dataset", actualStatusCode, 200)
	got := datasetJSONResponse(t, actualBody)
	dstest.CompareGoldenDatasetAndUpdateIfEnvVarSet(t, "testdata/expect/TestDatasetGet.test_ds.json", got)

	// Get csv body using "body.csv" suffix
	actualStatusCode, actualBody = APICall("/get/peer/test_ds/body.csv", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds", "selector": "body.csv"})
	expectBody := "city,pop,avg_age,in_usa\ntoronto,40000000,55.5,false\nnew york,8500000,44.4,true\nchicago,300000,44.4,true\nchatham,35000,65.25,true\nraleigh,250000,50.65,true\n"
	assertStatusCode(t, "get body.csv using suffix", actualStatusCode, 200)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Can get zip file
	actualStatusCode, _ = APICall("/get/peer/test_ds?format=zip", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds"})
	assertStatusCode(t, "get zip file", actualStatusCode, 200)

	// Can get a single component
	actualStatusCode, _ = APICall("/get/peer/test_ds/meta", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds", "selector": "meta"})
	assertStatusCode(t, "get meta component", actualStatusCode, 200)

	// Can get at an ipfs version
	actualStatusCode, _ = APICall("/get/peer/test_ds/at/mem/QmeTvt83npHg4HoxL8bp8yz5bmG88hUVvRc5k9taW8uxTr", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds", "fs": "mem", "hash": "QmeTvt83npHg4HoxL8bp8yz5bmG88hUVvRc5k9taW8uxTr"})
	assertStatusCode(t, "get at content-addressed version", actualStatusCode, 200)

	// Error 404 if ipfs version doesn't exist
	actualStatusCode, _ = APICall("/get/peer/test_ds/at/mem/QmissingEJUqFWNfdiPTPtxyba6wf86TmbQe1nifpZCRH6", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds", "fs": "mem", "hash": "QmissingEJUqFWNfdiPTPtxyba6wf86TmbQe1nifpZCRH6"})
	assertStatusCode(t, "get missing content-addressed version", actualStatusCode, 404)

	// Error 400 due to unknown component
	actualStatusCode, _ = APICall("/get/peer/test_ds/dunno", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds", "selector": "dunno"})
	assertStatusCode(t, "unknown component", actualStatusCode, 400)

	// Error 400 due to parse error of dsref
	actualStatusCode, _ = APICall("/get/peer/test+ds", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test+ds"})
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
		Data *dataset.Dataset
		Meta map[string]interface{}
	}{}
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		t.Fatal(err)
	}
	return res.Data
}

func TestGetCaseAndParamsFromRequest(t *testing.T) {
	var expectParams *lib.GetParams
	var expectCase string

	// get request with no headers
	r, _ := http.NewRequest("GET", "/get/peer/my_ds", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds", "selector": "meta"})
	expectParams = &lib.GetParams{
		Ref:      "peer/my_ds",
		Selector: "meta",
	}
	expectCase = ""
	gotCase, gotParams, err := getCaseAndParamsFromRequest(r)
	if err != nil {
		t.Fatalf("error getting case and params from basic get request: %s", err)
	}
	if diff := cmp.Diff(expectCase, gotCase); diff != "" {
		t.Errorf("request with no headers case mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("request with no headers params mismatch (-want +got):\n%s", diff)
	}

	// Construct a request with "Accept: text/csv"
	r, _ = http.NewRequest("GET", "", nil)
	r.Header.Add("Accept", "text/csv")
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds", "selector": "body"})
	expectParams = &lib.GetParams{
		Ref:      "peer/my_ds",
		Selector: "body",
		All:      true,
	}
	expectCase = "csv"
	gotCase, gotParams, err = getCaseAndParamsFromRequest(r)
	if err != nil {
		t.Fatalf("error getting case and params from get request with 'text/csv' header: %s", err)
	}
	if diff := cmp.Diff(expectCase, gotCase); diff != "" {
		t.Errorf("request with 'text/csv' header case mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("request with 'text/csv' header params mismatch (-want +got):\n%s", diff)
	}

	// Construct a request with format=csv and "Accept: text/csv", which is ok
	r, _ = http.NewRequest("GET", "?format=csv", nil)
	r.Header.Add("Accept", "text/csv")
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds", "selector": "body.csv"})
	expectParams = &lib.GetParams{
		Ref:      "peer/my_ds",
		Selector: "body",
		All:      true,
	}
	expectCase = "csv"
	gotCase, gotParams, err = getCaseAndParamsFromRequest(r)
	if err != nil {
		t.Fatalf("error getting case and params from get request with format=csv: %s", err)
	}
	if diff := cmp.Diff(expectCase, gotCase); diff != "" {
		t.Errorf("request with format=csv case mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("request with format=csv params mismatch (-want +gmt):\n%s", diff)
	}

	// Construct a request with format=json and "Accept: text/csv", which is an error
	r, _ = http.NewRequest("GET", "?format=json", nil)
	r.Header.Add("Accept", "text/csv")
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds"})

	_, _, err = getCaseAndParamsFromRequest(r)
	if err == nil {
		t.Error("expected to get an error, but did not get one")
	}
	expectErr := `format "json" conflicts with header "Accept: text/csv"`
	if expectErr != err.Error() {
		t.Errorf("error mismatch, expect: %q, got %q", expectErr, err)
	}

	// Construct a request with header "Accept: application/zip"
	r, _ = http.NewRequest("GET", "", nil)
	r.Header.Add("Accept", "application/zip")
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds"})
	expectParams = &lib.GetParams{
		Ref: "peer/my_ds",
	}
	expectCase = "zip"
	gotCase, gotParams, err = getCaseAndParamsFromRequest(r)
	if err != nil {
		t.Fatalf("error getting case and params from get request with 'application/zip' header: %s", err)
	}
	if diff := cmp.Diff(expectCase, gotCase); diff != "" {
		t.Errorf("request with 'application/zip' header case mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("request with 'application/zip' header params mismatch (-want +gbt):\n%s", diff)
	}

	// Construct a request with format=zip
	r, _ = http.NewRequest("GET", "?format=zip", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds"})
	expectParams = &lib.GetParams{
		Ref: "peer/my_ds",
	}
	expectCase = "zip"
	gotCase, gotParams, err = getCaseAndParamsFromRequest(r)
	if err != nil {
		t.Fatalf("error getting case and params from get request format=zip: %s", err)
	}
	if diff := cmp.Diff(expectCase, gotCase); diff != "" {
		t.Errorf("request with format=zip case mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("request with format=zip params mismatch (-want +gbt):\n%s", diff)
	}

	// Construct a request with dataset.json
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds", "selector": "dataset.json"})
	expectParams = &lib.GetParams{
		Ref: "peer/my_ds",
	}
	expectCase = "text/json"
	gotCase, gotParams, err = getCaseAndParamsFromRequest(r)
	if err != nil {
		t.Fatalf("error getting case and params from get request selector=dataset.json: %s", err)
	}
	if diff := cmp.Diff(expectCase, gotCase); diff != "" {
		t.Errorf("request with selector=dataset.json case mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("request with selector=dataset.json params mismatch (-want +gbt):\n%s", diff)
	}

	// Construct a request with dataset.json
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds", "selector": "meta.json"})
	expectParams = &lib.GetParams{
		Ref:      "peer/my_ds",
		Selector: "meta",
	}
	expectCase = "text/json"
	gotCase, gotParams, err = getCaseAndParamsFromRequest(r)
	if err != nil {
		t.Fatalf("error getting case and params from get request selector=meta.json: %s", err)
	}
	if diff := cmp.Diff(expectCase, gotCase); diff != "" {
		t.Errorf("request with selector=meta.json case mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("request with selector=meta.json params mismatch (-want +gbt):\n%s", diff)
	}

	// Construct a request with format pretty
	r, _ = http.NewRequest("GET", "?format=pretty", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds", "selector": "meta"})
	expectParams = &lib.GetParams{
		Ref:      "peer/my_ds",
		Selector: "meta",
	}
	expectCase = "pretty"
	gotCase, gotParams, err = getCaseAndParamsFromRequest(r)
	if err != nil {
		t.Fatalf("error getting case and params from get request format=pretty: %s", err)
	}
	if diff := cmp.Diff(expectCase, gotCase); diff != "" {
		t.Errorf("request with format=pretty case mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("request with format=pretty params mismatch (-want +gbt):\n%s", diff)
	}

	// Construct a request with dataset.json
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds", "selector": "meta.json"})
	expectParams = &lib.GetParams{
		Ref:      "peer/my_ds",
		Selector: "meta",
	}
	expectCase = "text/json"
	gotCase, gotParams, err = getCaseAndParamsFromRequest(r)
	if err != nil {
		t.Fatalf("error getting case and params from get request selector=meta.json: %s", err)
	}
	if diff := cmp.Diff(expectCase, gotCase); diff != "" {
		t.Errorf("request with selector=meta.json case mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("request with selector=meta.json params mismatch (-want +gbt):\n%s", diff)
	}
}

func mustSetMuxVarsOnRequest(t *testing.T, r *http.Request, muxVars map[string]string) *http.Request {
	r = mux.SetURLVars(r, muxVars)
	setRefStringFromMuxVars(r)
	if err := setMuxVarsToQueryParams(r); err != nil {
		t.Fatal(err)
	}
	return r
}
