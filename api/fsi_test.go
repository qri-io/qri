package api

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/qri/fsi"
)

func TestFSIHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	inst := newTestInstanceWithProfileFromNode(node)
	h := NewFSIHandlers(inst, false)

	// TODO (b5) - b/c of the way our API snapshotting we have to write these
	// folders to relative paths :( bad!
	checkoutDir := "fsi_tests/family_relationships"
	initDir := "fsi_tests/init_dir"
	if err := os.MkdirAll(initDir, os.ModePerm); err != nil {
		panic(err)
	}
	defer os.RemoveAll(filepath.Join("fsi_tests"))

	initCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", nil},
		{"POST", "/", nil},
		{"POST", fmt.Sprintf("/?filepath=%s", initDir), nil},
		{"POST", fmt.Sprintf("/?filepath=%s&name=api_test_init_dataset", initDir), nil},
		{"POST", fmt.Sprintf("/?filepath=%s&name=api_test_init_dataset&format=csv", initDir), nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "init", h.InitHandler(""), initCases, true)

	statusCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		// TODO (b5) - can't ask for an FSI-linked status b/c the responses change with
		// temp directory names
		{"GET", "/me/movies", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "status", h.StatusHandler(""), statusCases, true)

	checkoutCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/me/movies", nil},
		// TODO (b5) - can't ask for an FSI-linked status b/c the responses change with
		// temp directory names
		{"POST", fmt.Sprintf("/me/movies?dir=%s", checkoutDir), nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "checkout", h.CheckoutHandler(""), checkoutCases, true)
}

func TestNoHistory(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	inst := newTestInstanceWithProfileFromNode(node)

	initDir := "fsi_tests/init_dir"
	if err := os.MkdirAll(initDir, os.ModePerm); err != nil {
		panic(err)
	}
	defer os.RemoveAll(filepath.Join("fsi_tests"))

	// Create a linked dataset without saving, it has no versions in the repository
	f := fsi.NewFSI(node.Repo)
	ref, err := f.InitDataset(fsi.InitParams{
		Filepath: initDir,
		Name:     "test_ds",
		Format:   "csv",
	})
	if err != nil {
		t.Fatal(err)
	}

	if ref != "peer/test_ds" {
		t.Errorf("expected ref to be \"peer/test_ds\", got \"%s\"", ref)
	}

	dsHandler := NewDatasetHandlers(node, false)

	// Dataset with no history
	actualStatusCode, actualBody := APICall("/peer/test_ds", dsHandler.GetHandler)
	if actualStatusCode != 422 {
		t.Errorf("expected status code 422, got %d", actualStatusCode)
	}
	expectBody := `{ "meta": { "code": 422, "error": "no history" }, "data": null }`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	// Dataset with no history, but FSI working directory has contents
	actualStatusCode, actualBody = APICall("/peer/test_ds?fsi=true", dsHandler.GetHandler)
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody = `{"data":{"peername":"peer","name":"test_ds","fsiPath":"fsi_tests/init_dir","dataset":{"bodyPath":"fsi_tests/init_dir/body.csv","meta":{"keywords":[],"qri":"md:0"},"name":"test_ds","peername":"peer","qri":"ds:0","structure":{"format":"csv","qri":"st:0","schema":{"items":{"items":[{"title":"name","type":"string"},{"title":"describe","type":"string"},{"title":"quantity","type":"integer"}],"type":"array"},"type":"array"}}},"published":false},"meta":{"code":200}}`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	// Body with no history
	actualStatusCode, actualBody = APICall("/body/peer/test_ds", dsHandler.BodyHandler)
	if actualStatusCode != 422 {
		t.Errorf("expected status code 422, got %d", actualStatusCode)
	}
	expectBody = `{ "meta": { "code": 422, "error": "no history" }, "data": null }`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	// Body with no history, but FSI working directory has body
	actualStatusCode, actualBody = APICall("/body/peer/test_ds?fsi=true", dsHandler.BodyHandler)
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody = `{"data":{"path":"","data":[["one","two",3],["four","five",6]]},"meta":{"code":200},"pagination":{"nextUrl":"/body/peer/test_ds?fsi=true\u0026page=2"}}`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	fsiHandler := NewFSIHandlers(inst, false)

	// Status at version with no history
	actualStatusCode, actualBody = APICall("/status/peer/test_ds", fsiHandler.StatusHandler("/status"))
	if actualStatusCode != 422 {
		t.Errorf("expected status code 422, got %d", actualStatusCode)
	}
	expectBody = `{ "meta": { "code": 422, "error": "no history" }, "data": null }`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	// Status with no history, but FSI working directory has contents
	actualStatusCode, actualBody = APICall("/status/peer/test_ds?fsi=true", fsiHandler.StatusHandler("/status"))
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody = `{"data":[{"sourceFile":"fsi_tests/init_dir/meta.json","component":"meta","type":"add","message":""},{"sourceFile":"fsi_tests/init_dir/schema.json","component":"schema","type":"add","message":""},{"sourceFile":"body.csv","component":"body","type":"add","message":""}],"meta":{"code":200}}`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	logHandler := NewLogHandlers(node)

	// History with no history
	actualStatusCode, actualBody = APICall("/history/peer/test_ds", logHandler.LogHandler)
	if actualStatusCode != 422 {
		t.Errorf("expected status code 422, got %d", actualStatusCode)
	}
	expectBody = `{ "meta": { "code": 422, "error": "no history" }, "data": null }`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	// History with no history, still returns ErrNoHistory since this route ignores fsi param
	actualStatusCode, actualBody = APICall("/history/peer/test_ds?fsi=true", logHandler.LogHandler)
	if actualStatusCode != 422 {
		t.Errorf("expected status code 422, got %d", actualStatusCode)
	}
	expectBody = `{ "meta": { "code": 422, "error": "no history" }, "data": null }`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}
}

// APICall calls the api and returns the status code and body
func APICall(url string, hf http.HandlerFunc) (int, string) {
	req := httptest.NewRequest("GET", url, bytes.NewBuffer(nil))
	w := httptest.NewRecorder()
	hf(w, req)
	res := w.Result()
	statusCode := res.StatusCode
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	return statusCode, string(bodyBytes)
}
