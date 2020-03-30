package api

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
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
		{"OPTIONS", "/", nil},
		{"GET", "/", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "list", h.ListHandler, listCases, true)

	// TODO: Remove this case, update API snapshot.
	initCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/", mustFile(t, "testdata/newRequestFromURL.json")},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "init", h.SaveHandler, initCases, true)

	saveCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/", mustFile(t, "testdata/newRequestFromURL.json")},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "save", h.SaveHandler, saveCases, true)

	getCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/me/family_relationships", nil},
		{"GET", "/me/family_relationships/at/map/Qme7LVBp6hfi4Y5N29CXeXjpAqgT3fWtAmQWtZgjpQAZph", nil},
		{"GET", "/at/map/Qme7LVBp6hfi4Y5N29CXeXjpAqgT3fWtAmQWtZgjpQAZph", nil},
		// test that when fsi=true parameter doesn't affect the api response
		{"GET", "/me/family_relationships?fsi=true", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "get", h.GetHandler, getCases, true)

	bodyCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/body/me/family_relationships", nil},
		{"GET", "/body/me/family_relationships?download=true", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "body", h.BodyHandler, bodyCases, true)

	statsCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/stats/me/craigslist", nil},
		{"GET", "/stats/me/family_relationships/at/map/Qme7LVBp6hfi4Y5N29CXeXjpAqgT3fWtAmQWtZgjpQAZph", nil},
	}
	runHandlerTestCases(t, "stats", h.StatsHandler, statsCases, false)

	renameCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/rename", mustFile(t, "testdata/renameRequest.json")},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "rename", h.RenameHandler, renameCases, true)

	exportCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/export/me/cities", nil},
		{"GET", "/export/me/cities/at/map/QmPRjfgUFrH1GxBqujJ3sEvwV3gzHdux1j4g8SLyjbhwot", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "export", h.ZipDatasetHandler, exportCases, true)

	// TODO: Perhaps add an option to runHandlerTestCases to set Content-Type, then combin, truee
	// `runHandlerZipPostTestCases` with `runHandlerTestCases`, true.
	unpackCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/unpack/", mustFile(t, "testdata/exported.zip")},
	}
	runHandlerZipPostTestCases(t, "unpack", h.UnpackHandler, unpackCases)

	diffCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/?left_path=me/family_relationships&right_path=me/cities", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "diff", h.DiffHandler, diffCases, false)

	removeCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", nil},
		{"POST", "/remove/me/cities", nil},
		{"POST", "/remove/at/map/QmPRjfgUFrH1GxBqujJ3sEvwV3gzHdux1j4g8SLyjbhwot", nil},
	}
	runHandlerTestCases(t, "remove", h.RemoveHandler, removeCases, true)

	removeMimeCases := []handlerMimeMultipartTestCase{
		{"POST", "/remove/me/cities",
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
				"name":     "cities_dry_run",
				"dry_run":  "true",
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

	inst := newTestInstanceWithProfileFromNode(node)
	h := NewDatasetHandlers(inst, false)

	bodyPath := "testdata/cities/data.csv"

	// Save first version using a body path
	req := postJSONRequest(fmt.Sprintf("/save/?bodypath=%s&new=true", absolutePath(bodyPath)), "{}")
	w := httptest.NewRecorder()
	h.SaveHandler(w, req)
	bodyText := resultText(w)
	// Name is inferred from the body path
	expectText := `"name":"datacsv"`
	if !strings.Contains(bodyText, expectText) {
		t.Errorf("expected, body response to contain %q, not found. got %q", expectText, bodyText)
	}

	// Save a second time
	req = postJSONRequest(fmt.Sprintf("/save/?bodypath=%s&new=true", absolutePath(bodyPath)), "{}")
	w = httptest.NewRecorder()
	h.SaveHandler(w, req)
	bodyText = resultText(w)
	// Name is guaranteed to be unique
	expectText = `"name":"datacsv_1"`
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
