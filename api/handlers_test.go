package api

import (
	"encoding/json"
	"fmt"
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

func TestGetBodyCSVHandler(t *testing.T) {
	run := NewAPITestRunner(t)
	defer run.Delete()

	ds := dataset.Dataset{
		Name: "test_ds",
		Meta: &dataset.Meta{
			Title: "title one",
		},
	}
	run.SaveDataset(&ds, "testdata/cities/data.csv")

	// Get csv body using "body.csv" suffix
	actualStatusCode, actualBody := APICall("/get/peer/test_ds/body.csv", GetBodyCSVHandler(run.Inst), map[string]string{"username": "peer", "name": "test_ds"})
	expectBody := "city,pop,avg_age,in_usa\ntoronto,40000000,55.5,false\nnew york,8500000,44.4,true\nchicago,300000,44.4,true\nchatham,35000,65.25,true\nraleigh,250000,50.65,true\n"
	assertStatusCode(t, "get body.csv using suffix", actualStatusCode, 200)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
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

	// Can get csv body file using format
	actualStatusCode, _ = APICall("/get/peer/test_ds/body?format=csv", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds", "selector": "body"})
	assertStatusCode(t, "get csv file using format", actualStatusCode, 200)

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

func TestGetGetParamsFromRequest(t *testing.T) {
	var caseName string
	var expectParams *lib.GetParams
	var gotParams *lib.GetParams
	var expectErr error
	var r *http.Request

	// Construct a request with a bad ref
	caseName = "get request with bad ref"
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my+ds"})
	_, err := getGetParamsFromRequest(r)
	if err == nil {
		t.Errorf("case %q, expected error, get nil error", caseName)
	}

	// Construct a request with username "me"
	caseName = "get request with username 'me'"
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "me", "name": "my_ds"})
	expectErr = fmt.Errorf("username \"me\" not allowed")
	_, err = getGetParamsFromRequest(r)
	if expectErr.Error() != err.Error() {
		t.Errorf("case %q, expected error %q, got %q", caseName, expectErr, err)
	}

	// Construct a request with ref
	caseName = "get request with ref"
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds"})
	expectParams = &lib.GetParams{
		Ref: "peer/my_ds",
		All: true,
	}
	gotParams, err = getGetParamsFromRequest(r)
	if err != nil {
		t.Fatalf("case %q, error getting params from request: %s", caseName, err)
	}
	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("case %q, params mismatch (-want +gbt):\n%s", caseName, diff)
	}

	// Construct a request with a selector
	caseName = "get request with a selector"
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds", "selector": "meta"})
	expectParams = &lib.GetParams{
		Ref:      "peer/my_ds",
		Selector: "meta",
		All:      true,
	}
	gotParams, err = getGetParamsFromRequest(r)
	if err != nil {
		t.Fatalf("case %q, error getting params from request: %s", caseName, err)
	}
	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("case %q, params mismatch (-want +gbt):\n%s", caseName, diff)
	}

	// Construct a request with limit and offset
	caseName = "get request with limit and offset"
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds", "selector": "body", "limit": "0", "offset": "10"})
	expectParams = &lib.GetParams{
		Ref:      "peer/my_ds",
		Selector: "body",
		Limit:    0,
		Offset:   10,
	}
	gotParams, err = getGetParamsFromRequest(r)
	if err != nil {
		t.Fatalf("case %q, error getting params from request: %s", caseName, err)
	}
	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("case %q, params mismatch (-want +gbt):\n%s", caseName, diff)
	}

	// Construct a request with "all"
	caseName = "get request with 'all'"
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds", "selector": "body", "all": "true"})
	expectParams = &lib.GetParams{
		Ref:      "peer/my_ds",
		Selector: "body",
		All:      true,
	}
	gotParams, err = getGetParamsFromRequest(r)
	if err != nil {
		t.Fatalf("case %q, error getting params from request: %s", caseName, err)
	}
	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("case %q, params mismatch (-want +gbt):\n%s", caseName, diff)
	}
}

func TestValidateCSVRequest(t *testing.T) {
	var caseName string
	var expectErr error
	var r *http.Request
	var p *lib.GetParams

	// bad selector
	caseName = "selector is not body"
	r, _ = http.NewRequest("GET", "", nil)
	p = &lib.GetParams{}
	expectErr = fmt.Errorf("can only get csv of the body component, selector must be 'body'")
	err := validateCSVRequest(r, p)
	if expectErr.Error() != err.Error() {
		t.Errorf("case %q, expected error %q,  got %q", caseName, expectErr, err)
	}

	// add body selector to params
	p.Selector = "body"

	// bad format
	caseName = "bad format"
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "me", "name": "my_ds", "format": "json"})
	expectErr = fmt.Errorf("format \"json\" conflicts with requested body csv file")
	err = validateCSVRequest(r, p)
	if expectErr.Error() != err.Error() {
		t.Errorf("case %q, expected error %q, got %q", caseName, expectErr, err)
	}

	// valid request
	caseName = "valid request"
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds", "format": "csv"})
	err = validateCSVRequest(r, p)
	if err != nil {
		t.Errorf("case %q, unexpected error %q", caseName, err)
	}
}

func TestValidateZipRequest(t *testing.T) {
	var caseName string
	var expectErr error
	var r *http.Request
	var p *lib.GetParams

	// bad selector
	caseName = "selector is not empty"
	r, _ = http.NewRequest("GET", "", nil)
	p = &lib.GetParams{Selector: "meta"}
	expectErr = fmt.Errorf("can only get zip file of the entire dataset, got selector \"meta\"")
	err := validateZipRequest(r, p)
	if expectErr.Error() != err.Error() {
		t.Errorf("case %q, expected error %q,  got %q", caseName, expectErr, err)
	}

	// remove selector from params
	p.Selector = ""

	// bad format
	caseName = "bad format"
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "me", "name": "my_ds", "format": "json"})
	expectErr = fmt.Errorf("format %q conflicts with header %q", "json", "Accept: application/zip")
	err = validateZipRequest(r, p)
	if expectErr.Error() != err.Error() {
		t.Errorf("case %q, expected error %q, got %q", caseName, expectErr, err)
	}

	// valid request
	caseName = "valid request"
	r, _ = http.NewRequest("GET", "", nil)
	r = mustSetMuxVarsOnRequest(t, r, map[string]string{"username": "peer", "name": "my_ds", "format": "zip"})
	err = validateZipRequest(r, p)
	if err != nil {
		t.Errorf("case %q, unexpected error %q", caseName, err)
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
