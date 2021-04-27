package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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

	// incorrect http method
	actualStatusCode, actualBody = APICallWithParams("POST", "/get/peer/test_ds/body.csv", nil, GetBodyCSVHandler(run.Inst), nil)
	assertStatusCode(t, "get body.csv with incorrect http method", actualStatusCode, 404)

	// invalid request
	actualStatusCode, actualBody = APICall("/get/peer/test_ds/body.csv", GetBodyCSVHandler(run.Inst), map[string]string{"username": "peer", "name": "test_ds", "format": "json"})
	assertStatusCode(t, "get body.csv with incorrect http method", actualStatusCode, 400)
}

func TestDatasetGet(t *testing.T) {
	run := NewAPITestRunner(t)
	defer run.Delete()

	ds := dataset.Dataset{
		Name: "test_ds",
		Meta: &dataset.Meta{
			Title: "title one",
		},
		Readme: &dataset.Readme{
			ScriptBytes: []byte(`hello world`),
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

	// Can get a readme script
	actualStatusCode, _ = APICall("/get/peer/test_ds/readme.script", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds", "selector": "readme.script"})
	assertStatusCode(t, "get readme.script", actualStatusCode, 200)

	// Can get a single component
	actualStatusCode, _ = APICall("/get/peer/test_ds/meta", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds", "selector": "meta"})
	assertStatusCode(t, "get meta component", actualStatusCode, 200)

	// Can get at an ipfs version
	actualStatusCode, _ = APICall("/get/peer/test_ds/at/mem/QmX3Y2CG4DhZMHKTPAGPpLdwRPoWDjZLxAJwcikNYo8Tqa", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds", "fs": "mem", "hash": "QmX3Y2CG4DhZMHKTPAGPpLdwRPoWDjZLxAJwcikNYo8Tqa"})
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

func TestUnpackHandler(t *testing.T) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	text := []byte(`{ "meta": { "title": "hello world!" }}`)
	filename := "meta.json"
	f, err := zw.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Write(text)
	if err != nil {
		t.Fatal(err)
	}
	zw.Close()

	rr := bytes.NewReader(buf.Bytes())

	r := httptest.NewRequest("POST", "/unpack", rr)
	w := httptest.NewRecorder()

	hf := UnpackHandler("/unpack")
	hf(w, r)
	if w.Result().StatusCode != 200 {
		t.Errorf("%s", w.Body.String())
		t.Fatal(fmt.Errorf("expected unpack handler to return with 200 status code, returned with %d", w.Result().StatusCode))
	}

	r = httptest.NewRequest("GET", "/unpack", nil)
	w = httptest.NewRecorder()
	hf(w, r)
	if w.Result().StatusCode != 404 {
		t.Errorf("%s", w.Body.String())
		t.Fatal(fmt.Errorf("expected call to unpack handler with GET method to return status code 404, got %d", w.Result().StatusCode))
	}

	r = httptest.NewRequest("POST", "/unpack", nil)
	w = httptest.NewRecorder()
	hf(w, r)
	if w.Result().StatusCode != 500 {
		t.Errorf("%s", w.Body.String())
		t.Fatal(fmt.Errorf("expected call to unpack handler with GET method to return status code 500, got %d", w.Result().StatusCode))
	}
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

func TestExtensionToMimeType(t *testing.T) {
	cases := []struct {
		ext, expect string
	}{
		{".csv", "text/csv"},
		{".json", "application/json"},
		{".yaml", "application/x-yaml"},
		{".xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{".zip", "application/zip"},
		{".txt", "text/plain"},
		{".md", "text/x-markdown"},
		{".html", "text/html"},
		{"", ""},
	}
	for i, c := range cases {
		got := extensionToMimeType(c.ext)
		if c.expect != got {
			t.Errorf("case %d: expected %q got %q", i, c.expect, got)
		}
	}
}
