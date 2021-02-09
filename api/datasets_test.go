package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
)

func TestDatasetHandlers(t *testing.T) {
	run := NewAPITestRunner(t)
	defer run.Delete()

	// Create a mock data server. Can't move this into the testRunner, because we need to
	// ensure only this test is using the server's port "55555".
	s := newMockDataServer(t)
	defer s.Close()

	h := NewDatasetHandlers(run.Inst, false)

	listCases := []handlerTestCase{
		{"GET", "/", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "list", h.ListHandler, listCases, true)

	// TODO: Remove this case, update API snapshot.
	saveCases := []handlerTestCase{
		{"POST", "/", mustFile(t, "testdata/newRequestFromURL.json")},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "init", h.SaveHandler, saveCases, true)

	getCases := []handlerTestCase{
		{"GET", "/get/peer/family_relationships", nil},
		{"GET", "/get/peer/family_relationships/at/map/Qme7LVBp6hfi4Y5N29CXeXjpAqgT3fWtAmQWtZgjpQAZph", nil},
		{"GET", "/get/at/map/Qme7LVBp6hfi4Y5N29CXeXjpAqgT3fWtAmQWtZgjpQAZph", nil},
		// test that when fsi=true parameter doesn't affect the api response
		{"GET", "/get/peer/family_relationships?fsi=true", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "get", h.GetHandler, getCases, true)

	bodyCases := []handlerTestCase{
		{"GET", "/get/peer/family_relationships?component=body", nil},
		// TODO(arqu): broken, expecing object and not array response
		// {"GET", "/get/peer/family_relationships?component=body&download=true", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "body", h.GetHandler, bodyCases, true)

	statsCases := []handlerTestCase{
		{"GET", "/stats/peer/craigslist", nil},
		{"GET", "/stats/peer/family_relationships/at/map/Qme7LVBp6hfi4Y5N29CXeXjpAqgT3fWtAmQWtZgjpQAZph", nil},
	}
	runHandlerTestCases(t, "stats", h.StatsHandler, statsCases, false)

	renameCases := []handlerTestCase{
		{"POST", "/rename", mustFile(t, "testdata/renameRequest.json")},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "rename", h.RenameHandler, renameCases, true)

	// TODO: Perhaps add an option to runHandlerTestCases to set Content-Type, then combin, truee
	// `runHandlerZipPostTestCases` with `runHandlerTestCases`, true.
	unpackCases := []handlerTestCase{
		{"POST", "/unpack/", mustFile(t, "testdata/exported.zip")},
	}
	runHandlerZipPostTestCases(t, "unpack", h.UnpackHandler, unpackCases)

	diffCases := []handlerTestCase{
		{"GET", "/?left_path=peer/family_relationships&right_path=peer/cities", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "diff", h.DiffHandler, diffCases, false)

	removeCases := []handlerTestCase{
		{"GET", "/", nil},
		{"POST", "/remove/peer/cities", nil},
		{"POST", "/remove/at/map/QmPRjfgUFrH1GxBqujJ3sEvwV3gzHdux1j4g8SLyjbhwot", nil},
	}
	runHandlerTestCases(t, "remove", h.RemoveHandler, removeCases, true)

	removeMimeCases := []handlerMimeMultipartTestCase{
		{"POST", "/remove/peer/cities",
			map[string]string{},
			map[string]string{},
		},
	}
	runMimeMultipartHandlerTestCases(t, "remove mime/multipart", h.RemoveHandler, removeMimeCases)

	newMimeCases := []handlerMimeMultipartTestCase{
		{"POST", "/save",
			map[string]string{
				"body":      "testdata/cities/data.csv",
				"structure": "testdata/cities/structure.json",
				"metadata":  "testdata/cities/meta.json",
			},
			map[string]string{
				"peername": "peer",
				"name":     "cities",
				"private":  "true",
			},
		},
		{"POST", "/save",
			map[string]string{
				"body": "testdata/cities/data.csv",
				"file": "testdata/cities/init_dataset.json",
			},
			map[string]string{
				"peername": "peer",
				"name":     "cities",
			},
		},
		{"POST", "/save",
			map[string]string{
				"body":      "testdata/cities/data.csv",
				"structure": "testdata/cities/structure.json",
				"metadata":  "testdata/cities/meta.json",
			},
			map[string]string{
				"peername": "peer",
				"name":     "cities_2",
			},
		},
	}
	runMimeMultipartHandlerTestCases(t, "save mime/multipart", h.SaveHandler, newMimeCases)
}

func newMockDataServer(t *testing.T) *httptest.Server {
	mockData := []byte(`Parent Identifier,Student Identifier
1001,1002
1010,1020
`)
	mockDataServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(mockData)
	}))
	l, err := net.Listen("tcp", ":55555")
	if err != nil {
		t.Fatal(err.Error())
	}
	mockDataServer.Listener = l
	mockDataServer.Start()
	return mockDataServer
}

func TestSaveWithInferredNewName(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inst := newTestInstanceWithProfileFromNode(ctx, node)
	h := NewDatasetHandlers(inst, false)

	bodyPath := "testdata/cities/data.csv"

	// Save first version using a body path
	req := postJSONRequest(fmt.Sprintf("/save/?bodypath=%s&new=true", absolutePath(bodyPath)), "{}")
	w := httptest.NewRecorder()
	h.SaveHandler(w, req)
	bodyText := resultText(w)
	// Name is inferred from the body path
	expectText := `"name":"data"`
	if !strings.Contains(bodyText, expectText) {
		t.Errorf("expected, body response to contain %q, not found. got %q", expectText, bodyText)
	}

	// Save a second time
	req = postJSONRequest(fmt.Sprintf("/save/?bodypath=%s&new=true", absolutePath(bodyPath)), "{}")
	w = httptest.NewRecorder()
	h.SaveHandler(w, req)
	bodyText = resultText(w)
	// Name is guaranteed to be unique
	expectText = `"name":"data_2"`
	if !strings.Contains(bodyText, expectText) {
		t.Errorf("expected, body response to contain %q, not found. got %q", expectText, bodyText)
	}
}

func postJSONRequest(url, jsonBody string) *http.Request {
	req := httptest.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonBody)))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func absolutePath(text string) string {
	res, _ := filepath.Abs(text)
	return res
}

func resultText(rec *httptest.ResponseRecorder) string {
	res := rec.Result()
	bytes, _ := ioutil.ReadAll(res.Body)
	return string(bytes)
}

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
					All:      true,
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
			"download all of the body as csv",
			"/get/peer/my_ds?download=true&format=csv&component=body&all=true",
			&GetReqArgs{
				Ref:         dsref.MustParse("peer/my_ds"),
				RawDownload: true,
				Params: lib.GetParams{
					Refstr:   "peer/my_ds",
					Format:   "csv",
					Selector: "body",
					Limit:    100,
					All:      true,
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

func TestParseGetReqArgsAcceptHeader(t *testing.T) {
	// Construct a request with "Accept: text/csv"
	r, _ := http.NewRequest("GET", "/get/peer/my_ds", nil)
	r.Header.Add("Accept", "text/csv")
	reqPath := trimGetOrBodyPrefix(r.URL.Path)
	args, err := parseGetReqArgs(r, reqPath)
	if err != nil {
		t.Fatal(err)
	}
	expectArgs := &GetReqArgs{
		Ref: dsref.MustParse("peer/my_ds"),
		Params: lib.GetParams{
			Refstr:   "peer/my_ds",
			Selector: "body",
			Format:   "csv",
			Limit:    100,
		},
		RawDownload: true,
	}
	if diff := cmp.Diff(expectArgs, args); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Construct a request with format=csv and "Accept: text/csv", which is ok
	r, _ = http.NewRequest("GET", "/get/peer/my_ds?format=csv", nil)
	r.Header.Add("Accept", "text/csv")
	reqPath = trimGetOrBodyPrefix(r.URL.Path)
	args, err = parseGetReqArgs(r, reqPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectArgs, args); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	// Construct a request with format=json and "Accept: text/csv", which is an error
	r, _ = http.NewRequest("GET", "/get/peer/my_ds?format=json", nil)
	r.Header.Add("Accept", "text/csv")
	reqPath = trimGetOrBodyPrefix(r.URL.Path)
	args, err = parseGetReqArgs(r, reqPath)
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
	gotStatusCode, gotBodyString := APICall("/get/peer/test_ds?format=zip", dsHandler.GetHandler)
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

	actualStatusCode, actualBody := APICall("/get/peer/test_ds", dsHandler.GetHandler)
	assertStatusCode(t, "get dataset", actualStatusCode, 200)
	got := datasetJSONResponse(t, actualBody)
	dstest.CompareGoldenDatasetAndUpdateIfEnvVarSet(t, "testdata/expect/TestDatasetGet.test_ds.json", got)

	// Get csv body using "body.csv" suffix
	actualStatusCode, actualBody = APICall("/get/peer/test_ds/body.csv", dsHandler.GetHandler)
	expectBody := "city,pop,avg_age,in_usa\ntoronto,40000000,55.5,false\nnew york,8500000,44.4,true\nchicago,300000,44.4,true\nchatham,35000,65.25,true\nraleigh,250000,50.65,true\n"
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
	actualStatusCode, _ = APICall("/get/peer/test_ds/at/mem/QmeTvt83npHg4HoxL8bp8yz5bmG88hUVvRc5k9taW8uxTr", dsHandler.GetHandler)
	assertStatusCode(t, "get at content-addressed version", actualStatusCode, 200)

	// Error 404 if ipfs version doesn't exist
	actualStatusCode, _ = APICall("/get/peer/test_ds/at/mem/QmissingEJUqFWNfdiPTPtxyba6wf86TmbQe1nifpZCRH6", dsHandler.GetHandler)
	assertStatusCode(t, "get missing content-addressed version", actualStatusCode, 404)

	// Error 400 due to format=csv without download=true
	actualStatusCode, _ = APICall("/get/peer/test_ds?format=csv", dsHandler.GetHandler)
	assertStatusCode(t, "using format=csv", actualStatusCode, 400)

	// Error 400 due to unknown component
	actualStatusCode, _ = APICall("/get/peer/test_ds?component=dunno", dsHandler.GetHandler)
	assertStatusCode(t, "unknown component", actualStatusCode, 400)

	// Error 400 due to parse error of dsref
	actualStatusCode, _ = APICall("/get/peer/test+ds", dsHandler.GetHandler)
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
