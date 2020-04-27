package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	reporef "github.com/qri-io/qri/repo/ref"
)

func TestFSIHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	inst := newTestInstanceWithProfileFromNode(node)
	h := NewFSIHandlers(inst, false)

	// TODO (b5) - b/c of the way our API snapshotting we have to write these
	// folders to relative paths :( bad!
	_ = "fsi_tests/family_relationships"
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
		// TODO(dlong): Disabled, contains local file paths.
		//{"POST", fmt.Sprintf("/?filepath=%s&name=api_test_init_dataset&format=csv", initDir), nil},
		//{"DELETE", "/", nil},
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

	whatChangedCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		// TODO (b5) - can't ask for an FSI-linked status b/c the responses change with
		// temp directory names
		{"GET", "/me/movies", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "whatchanged", h.WhatChangedHandler(""), whatChangedCases, true)

	checkoutCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/me/movies", nil},
		// TODO (b5) - can't ask for an FSI-linked status b/c the responses change with
		// temp directory names
		//{"POST", fmt.Sprintf("/me/movies?dir=%s", checkoutDir), nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "checkout", h.CheckoutHandler(""), checkoutCases, true)
}

type APITestRunner struct {
	Node         *p2p.QriNode
	NodeTeardown func()
	Inst         *lib.Instance
	TmpDir       string
	WorkDir      string
	PrevXformVer string
}

func NewAPITestRunner(t *testing.T) *APITestRunner {
	run := APITestRunner{}
	run.Node, run.NodeTeardown = newTestNode(t)
	run.Inst = newTestInstanceWithProfileFromNode(run.Node)

	tmpDir, err := ioutil.TempDir("", "api_test")
	if err != nil {
		t.Fatal(err)
	}
	run.TmpDir = tmpDir

	run.PrevXformVer = APIVersion
	APIVersion = "test_version"

	return &run
}

func (r *APITestRunner) Delete() {
	os.RemoveAll(r.TmpDir)
	APIVersion = r.PrevXformVer
	r.NodeTeardown()
}

func (r *APITestRunner) MustMakeWorkDir(t *testing.T, name string) string {
	r.WorkDir = filepath.Join(r.TmpDir, name)
	if err := os.MkdirAll(r.WorkDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	return r.WorkDir
}

// TODO (ramfox): this test should be split for each endpoint:
// getHandler
// rootHandler
// bodyHandler
// logHandler
// statusHandler
// each test should test responses for a dataset with no history, fsi=true and fsi=false
func TestNoHistory(t *testing.T) {
	run := NewAPITestRunner(t)
	defer run.Delete()

	subDir := "fsi_init_dir"
	workDir := run.MustMakeWorkDir(t, subDir)

	// Create a linked dataset without saving, it has no versions in the repository
	ref, err := run.Inst.FSI().InitDataset(fsi.InitParams{
		Dir:    workDir,
		Name:   "test_ds",
		Format: "csv",
	})
	if err != nil {
		t.Fatal(err)
	}

	if ref != "peer/test_ds" {
		t.Errorf("expected ref to be \"peer/test_ds\", got \"%s\"", ref)
	}

	// Get mtimes for the component files
	st, _ := os.Stat(filepath.Join(workDir, "meta.json"))
	metaMtime := st.ModTime().Format(time.RFC3339)
	st, _ = os.Stat(filepath.Join(workDir, "body.csv"))
	bodyMtime := st.ModTime().Format(time.RFC3339)
	st, _ = os.Stat(filepath.Join(workDir, "structure.json"))
	structureMtime := st.ModTime().Format(time.RFC3339)

	dsHandler := NewDatasetHandlers(run.Inst, false)

	// Expected response for dataset head, regardless of fsi parameter
	expectBody := `{"data":{"peername":"peer","name":"test_ds","fsiPath":"fsi_init_dir","dataset":{"bodyPath":"fsi_init_dir/body.csv","meta":{"qri":"md:0"},"name":"test_ds","peername":"peer","qri":"ds:0","structure":{"format":"csv","qri":"st:0"}},"published":false},"meta":{"code":200}}`

	// Dataset with a link to the filesystem, but no history and the api request says fsi=false
	gotStatusCode, gotBodyString := APICall("/peer/test_ds", dsHandler.GetHandler)
	if gotStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", gotStatusCode)
	}
	actualBody := strings.Replace(gotBodyString, workDir, subDir, -1)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("expected body %v, got %v\ndiff:%v", expectBody, actualBody, diff)
	}

	// Dataset with a link to the filesystem, but no history and the api request says fsi=true
	gotStatusCode, gotBodyString = APICall("/peer/test_ds?fsi=true", dsHandler.GetHandler)
	if gotStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", gotStatusCode)
	}
	actualBody = strings.Replace(gotBodyString, workDir, subDir, -1)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("api response (-want +got):\n%s", diff)
	}

	// Expected response for body of the dataset
	expectBody = `{"data":{"path":"fsi_init_dir/body.csv","data":[["one","two",3],["four","five",6]]},"meta":{"code":200},"pagination":{"nextUrl":"/body/peer/test_ds?page=2"}}`

	// Body with no history, but fsi working directory has body
	gotStatusCode, gotBodyString = APICall("/body/peer/test_ds", dsHandler.BodyHandler)
	if gotStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", gotStatusCode)
	}
	actualBody = strings.Replace(gotBodyString, workDir, subDir, -1)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("expected body %v, got %v\ndiff:%v", expectBody, actualBody, diff)
	}

	// Body with no history, but fsi working directory has body
	gotStatusCode, gotBodyString = APICall("/body/peer/test_ds?fsi=true", dsHandler.BodyHandler)
	if gotStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", gotStatusCode)
	}
	actualBody = strings.Replace(gotBodyString, workDir, subDir, -1)
	actualBody = strings.Replace(actualBody, `fsi=true\u0026`, "", -1)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("expected body %v, got %v\ndiff:%v", expectBody, actualBody, diff)
	}

	fsiHandler := NewFSIHandlers(run.Inst, false)

	// Expected response for status of the dataset
	templateBody := `{"data":[{"sourceFile":"fsi_init_dir/meta.json","component":"meta","type":"add","message":"","mtime":"%s"},{"sourceFile":"fsi_init_dir/structure.json","component":"structure","type":"add","message":"","mtime":"%s"},{"sourceFile":"fsi_init_dir/body.csv","component":"body","type":"add","message":"","mtime":"%s"}],"meta":{"code":200}}`
	expectBody = fmt.Sprintf(templateBody, metaMtime, structureMtime, bodyMtime)

	// Status at version with no history
	gotStatusCode, gotBodyString = APICall("/status/peer/test_ds", fsiHandler.StatusHandler("/status"))
	if gotStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", gotStatusCode)
	}
	actualBody = strings.Replace(gotBodyString, workDir, subDir, -1)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("expected body %v, got %v\ndiff:%v", expectBody, actualBody, diff)
	}

	// Status with no history, but FSI working directory has contents
	gotStatusCode, gotBodyString = APICall("/status/peer/test_ds?fsi=true", fsiHandler.StatusHandler("/status"))
	if gotStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", gotStatusCode)
	}
	actualBody = strings.Replace(gotBodyString, workDir, subDir, -1)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("api response (-want +got):\n%s", diff)
	}

	logHandler := NewLogHandlers(run.Inst)

	expectNoHistoryBody := `{
  "meta": {
    "code": 422,
    "error": "repo: no history"
  }
}`

	// History with no history
	gotStatusCode, gotBodyString = APICall("/history/peer/test_ds", logHandler.LogHandler)
	if gotStatusCode != 422 {
		t.Errorf("expected status code 422, got %d", gotStatusCode)
	}
	if diff := cmp.Diff(expectNoHistoryBody, gotBodyString); diff != "" {
		t.Errorf("expected body %v, got %v\ndiff:%v", expectNoHistoryBody, gotBodyString, diff)
	}

	// History with no history, still returns ErrNoHistory since this route ignores fsi param
	gotStatusCode, gotBodyString = APICall("/history/peer/test_ds?fsi=true", logHandler.LogHandler)
	if gotStatusCode != 422 {
		t.Errorf("expected status code 422, got %d", gotStatusCode)
	}
	if diff := cmp.Diff(expectNoHistoryBody, gotBodyString); diff != "" {
		t.Errorf("expected body %v, got %v\ndiff:%v", expectNoHistoryBody, gotBodyString, diff)
	}
}

func TestFSIWrite(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	inst := newTestInstanceWithProfileFromNode(node)

	tmpDir := os.TempDir()
	workSubdir := "write_test"
	workDir := filepath.Join(tmpDir, workSubdir)
	// Don't create the work directory, it must not exist for checkout to work. Remove if it
	// already exists.
	_ = os.RemoveAll(workDir)

	dr := lib.NewDatasetRequests(node, nil)
	fsiHandler := NewFSIHandlers(inst, false)

	// Save version 1
	saveParams := lib.SaveParams{
		Ref: "me/write_test",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "title one",
			},
		},
		BodyPath: "testdata/cities/data.csv",
	}
	res := reporef.DatasetRef{}
	if err := dr.Save(&saveParams, &res); err != nil {
		t.Fatal(err)
	}

	// Checkout the dataset
	actualStatusCode, actualBody := APICallWithParams(
		"POST",
		"/checkout/peer/write_test",
		map[string]string{
			"dir": workDir,
		},
		fsiHandler.CheckoutHandler("/checkout"))
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody := `{"data":"","meta":{"code":200}}`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	status, strRes := JSONAPICallWithBody("POST", "/me/write_test", &dataset.Dataset{Meta: &dataset.Meta{Title: "oh hai there"}}, fsiHandler.WriteHandler(""))

	if status != http.StatusOK {
		t.Errorf("status code mismatch. expected: %d, got: %d", http.StatusOK, status)
	}

	exp := struct {
		Data []struct {
			// ignore mtime & path fields by only deserializing component & type from JSON
			Component string
			Type      string
		}
	}{
		Data: []struct {
			Component string
			Type      string
		}{
			{Component: "meta", Type: "modified"},
			{Component: "structure", Type: "unmodified"},
			{Component: "body", Type: "unmodified"},
		},
	}

	got := struct {
		Data []struct {
			Component string
			Type      string
		}
	}{}
	if err := json.NewDecoder(strings.NewReader(strRes)).Decode(&got); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(exp, got); diff != "" {
		t.Errorf("response data mistmach (-want +got):\n%s", diff)
	}
}

func TestCheckoutAndRestore(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	inst := newTestInstanceWithProfileFromNode(node)

	tmpDir := os.TempDir()
	workSubdir := "fsi_checkout_restore"
	workDir := filepath.Join(tmpDir, workSubdir)
	// Don't create the work directory, it must not exist for checkout to work. Remove if it
	// already exists.
	_ = os.RemoveAll(workDir)

	dr := lib.NewDatasetRequests(node, nil)

	// Save version 1
	saveParams := lib.SaveParams{
		Ref: "me/fsi_checkout_restore",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "title one",
			},
		},
		BodyPath: "testdata/cities/data.csv",
	}
	res := reporef.DatasetRef{}
	if err := dr.Save(&saveParams, &res); err != nil {
		t.Fatal(err)
	}

	// Save the path from reference for later.
	// TODO(dlong): Support full dataset refs, not just the path.
	pos := strings.Index(res.String(), "/map/")
	ref1 := res.String()[pos:]

	// Save version 2 with a different title
	saveParams = lib.SaveParams{
		Ref: "me/fsi_checkout_restore",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "title two",
			},
		},
	}
	if err := dr.Save(&saveParams, &res); err != nil {
		t.Fatal(err)
	}

	fsiHandler := NewFSIHandlers(inst, false)

	// Checkout the dataset
	actualStatusCode, actualBody := APICallWithParams(
		"POST",
		"/checkout/peer/fsi_checkout_restore",
		map[string]string{
			"dir": workDir,
		},
		fsiHandler.CheckoutHandler("/checkout"))
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody := `{"data":"","meta":{"code":200}}`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	// Read meta.json, should have "title two" as the meta title
	metaContents, err := ioutil.ReadFile(filepath.Join(workDir, "meta.json"))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expectContents := "{\n \"qri\": \"md:0\",\n \"title\": \"title two\"\n}"
	if diff := cmp.Diff(expectContents, string(metaContents)); diff != "" {
		t.Errorf("meta.json contents (-want +got):\n%s", diff)
	}

	// Overwrite meta so it has a different title
	if err = ioutil.WriteFile(filepath.Join(workDir, "meta.json"), []byte(`{"title": "hello"}`), os.ModePerm); err != nil {
		t.Fatalf(err.Error())
	}

	// Get mtimes for the component files
	st, _ := os.Stat(filepath.Join(workDir, "meta.json"))
	metaMtime := st.ModTime().Format(time.RFC3339)
	st, _ = os.Stat(filepath.Join(workDir, "structure.json"))
	structureMtime := st.ModTime().Format(time.RFC3339)
	st, _ = os.Stat(filepath.Join(workDir, "body.csv"))
	bodyMtime := st.ModTime().Format(time.RFC3339)

	// Status should show that meta is modified
	actualStatusCode, actualBody = APICall("/status/peer/fsi_checkout_restore?fsi=true", fsiHandler.StatusHandler("/status"))
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	// Handle temporary directory by replacing the temp part with a shorter string.
	resultBody := strings.Replace(actualBody, workDir, "", -1)
	templateBody := `{"data":[{"sourceFile":"/meta.json","component":"meta","type":"modified","message":"","mtime":"%s"},{"sourceFile":"/structure.json","component":"structure","type":"unmodified","message":"","mtime":"%s"},{"sourceFile":"/body.csv","component":"body","type":"unmodified","message":"","mtime":"%s"}],"meta":{"code":200}}`
	expectBody = fmt.Sprintf(templateBody, metaMtime, structureMtime, bodyMtime)
	if diff := cmp.Diff(expectBody, resultBody); diff != "" {
		t.Errorf("api response (-want +got):\n%s", diff)
	}

	// Restore the meta component
	actualStatusCode, actualBody = APICallWithParams(
		"POST",
		"/restore/peer/fsi_checkout_restore",
		map[string]string{
			"component": "meta",
		},
		fsiHandler.RestoreHandler("/restore"))
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody = `{"data":"","meta":{"code":200}}`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	// Read meta.json, should once again have "title two" as the meta title
	metaContents, err = ioutil.ReadFile(filepath.Join(workDir, "meta.json"))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expectContents = "{\n \"qri\": \"md:0\",\n \"title\": \"title two\"\n}"
	if diff := cmp.Diff(expectContents, string(metaContents)); diff != "" {
		t.Errorf("meta.json contents (-want +got):\n%s", diff)
	}

	// Restore the previous version of the dataset
	actualStatusCode, actualBody = APICallWithParams(
		"POST",
		"/restore/peer/fsi_checkout_restore",
		map[string]string{
			// TODO(dlong): Have to pass "dir" to this method. In the test, the ref does
			// not have an FSIPath. Might be because we're using /map/, not sure.
			"dir":  workDir,
			"path": ref1,
		},
		fsiHandler.RestoreHandler("/restore"))
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody = `{"data":"","meta":{"code":200}}`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	// Read meta.json, should now have "title one" as the meta title
	metaContents, err = ioutil.ReadFile(filepath.Join(workDir, "meta.json"))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expectContents = "{\n \"qri\": \"md:0\",\n \"title\": \"title one\"\n}"
	if diff := cmp.Diff(expectContents, string(metaContents)); diff != "" {
		t.Errorf("meta.json contents (-want +got):\n%s", diff)
	}
}

// APICall calls the api and returns the status code and body
func APICall(url string, hf http.HandlerFunc) (int, string) {
	return APICallWithParams("GET", url, nil, hf)
}

// APICallWithParams calls the api and returns the status code and body
func APICallWithParams(method, reqURL string, params map[string]string, hf http.HandlerFunc) (int, string) {
	// Add parameters from map
	reqParams := url.Values{}
	if params != nil {
		for key := range params {
			reqParams.Set(key, params[key])
		}
	}
	req := httptest.NewRequest(method, reqURL, strings.NewReader(reqParams.Encode()))
	// Set form-encoded header so server will find the parameters
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(reqParams.Encode())))
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

func JSONAPICallWithBody(method, reqURL string, data interface{}, hf http.HandlerFunc) (int, string) {
	enc, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	req := httptest.NewRequest(method, reqURL, bytes.NewReader(enc))
	// Set form-encoded header so server will find the parameters
	req.Header.Add("Content-Type", "application/json")
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
