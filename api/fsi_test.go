package api

import (
	"bytes"
	"context"
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
	"github.com/gorilla/mux"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/lib"
	qhttp "github.com/qri-io/qri/lib/http"
)

func TestFSIHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inst := newTestInstanceWithProfileFromNode(ctx, node)

	// TODO (b5) - b/c of the way our API snapshotting we have to write these
	// folders to relative paths :( bad!
	_ = "fsi_tests/family_relationships"
	initDir := "fsi_tests/init_dir"
	if err := os.MkdirAll(initDir, os.ModePerm); err != nil {
		panic(err)
	}
	defer os.RemoveAll(filepath.Join("fsi_tests"))

	initHandler := func(w http.ResponseWriter, r *http.Request) {
		lib.NewHTTPRequestHandler(inst, "fsi.init").ServeHTTP(w, r)
	}
	body := []byte(fmt.Sprintf(`{"username":"me","name":"api_test_init_dataset","targetDir":%q,"format":"csv"}`, initDir))
	initCases := []handlerTestCase{
		{http.MethodPost, "/", nil, nil},
		{http.MethodPost, fmt.Sprintf("/me/api_test_init_dataset?targetdir=%s&format=csv", initDir), body, nil},
		{http.MethodPost, fmt.Sprintf("/me/api_test_init_dataset?targetdir=%s&format=csv", initDir), body, nil},
	}
	runHandlerTestCases(t, "init", initHandler, initCases, true)

	checkoutHandler := func(w http.ResponseWriter, r *http.Request) {
		lib.NewHTTPRequestHandler(inst, "fsi.checkout").ServeHTTP(w, r)
	}
	checkoutCases := []handlerTestCase{
		{http.MethodPost, "/me/movies", nil, nil},
		// TODO (b5) - can't ask for an FSI-linked status b/c the responses change with
		// temp directory names
		//{http.MethodPost, fmt.Sprintf("/me/movies?dir=%s", checkoutDir), nil},
	}
	runHandlerTestCases(t, "checkout", checkoutHandler, checkoutCases, true)
}

// TODO (ramfox): this test should be split for each endpoint:
// getHandler
// rootHandler
// bodyHandler
// logHandler
// statusHandler
// each test should test responses for a dataset with no history, fsi=true and fsi=false
func TestNoHistory(t *testing.T) {
	ctx := context.Background()
	run := NewAPITestRunner(t)
	defer run.Delete()

	subDir := "fsi_init_dir"
	workDir := run.MustMakeWorkDir(t, subDir)

	// Create a linked dataset without saving, it has no versions in the repository
	ref, err := run.Inst.FSI().InitDataset(ctx, fsi.InitParams{
		TargetDir: workDir,
		Name:      "test_ds",
		Format:    "csv",
	})
	if err != nil {
		t.Fatal(err)
	}

	if ref.Human() != "peer/test_ds" {
		t.Errorf("expected ref to be \"peer/test_ds\", got \"%s\"", ref)
	}

	// Get mtimes for the component files
	st, _ := os.Stat(filepath.Join(workDir, "meta.json"))
	metaMtime := st.ModTime().Format(time.RFC3339)
	st, _ = os.Stat(filepath.Join(workDir, "body.csv"))
	bodyMtime := st.ModTime().Format(time.RFC3339)
	st, _ = os.Stat(filepath.Join(workDir, "structure.json"))
	structureMtime := st.ModTime().Format(time.RFC3339)

	// Expected response for dataset head, regardless of fsi parameter
	expectBody := `{"data":{"bodyPath":"fsi_init_dir/body.csv","id":"6yn4jmjjsndpwf4qllvjcunrzni3o3vcejlkvdsvhayfcvrgrpsq","meta":{"qri":"md:0"},"name":"test_ds","peername":"peer","qri":"ds:0","structure":{"format":"csv","qri":"st:0"}},"meta":{"code":200}}`

	// Dataset with a link to the filesystem, but no history and the api request says fsi=false
	gotStatusCode, gotBodyString := APICall("/get/peer/test_ds", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds"})
	if gotStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", gotStatusCode)
	}
	actualBody := strings.Replace(gotBodyString, workDir, subDir, -1)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("expected body %v, got %v\ndiff:%v", expectBody, actualBody, diff)
	}

	// Dataset with a link to the filesystem, but no history and the api request says fsi=true
	gotStatusCode, gotBodyString = APICall("/get/peer/test_ds?fsi=true", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds"})
	if gotStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", gotStatusCode)
	}
	actualBody = strings.Replace(gotBodyString, workDir, subDir, -1)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("api response (-want +got):\n%s", diff)
	}

	// Expected response for body of the dataset
	expectBody = `{"data":[["one","two",3],["four","five",6]],"meta":{"code":200}}`

	// Body with no history, but fsi working directory has body
	gotStatusCode, gotBodyString = APICall("/get/peer/test_ds/body", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds", "selector": "body"})
	if gotStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", gotStatusCode)
	}
	actualBody = strings.Replace(gotBodyString, workDir, subDir, -1)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("expected body %v, got %v\ndiff:%v", expectBody, actualBody, diff)
	}

	// Body with no history, but fsi working directory has body
	gotStatusCode, gotBodyString = APICall("/get/peer/test_ds/body&fsi=true", GetHandler(run.Inst, ""), map[string]string{"username": "peer", "name": "test_ds", "selector": "body"})
	if gotStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", gotStatusCode)
	}
	actualBody = strings.Replace(gotBodyString, workDir, subDir, -1)
	actualBody = strings.Replace(actualBody, `\u0026fsi=true`, "", -1)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("expected body %v, got %v\ndiff:%v", expectBody, actualBody, diff)
	}

	statusHandler := func(w http.ResponseWriter, r *http.Request) {
		lib.NewHTTPRequestHandler(run.Inst, "fsi.status").ServeHTTP(w, r)
	}

	// Expected response for status of the dataset
	templateBody := `{"data":[{"sourceFile":"fsi_init_dir/meta.json","component":"meta","type":"add","message":"","mtime":"%s"},{"sourceFile":"fsi_init_dir/structure.json","component":"structure","type":"add","message":"","mtime":"%s"},{"sourceFile":"fsi_init_dir/body.csv","component":"body","type":"add","message":"","mtime":"%s"}],"meta":{"code":200}}`
	expectBody = fmt.Sprintf(templateBody, metaMtime, structureMtime, bodyMtime)

	// Status at version with no history
	body := map[string]string{"ref": "peer/test_ds"}
	gotStatusCode, gotBodyString = JSONAPICallWithBody(http.MethodPost, "/status", body, statusHandler, nil)
	if gotStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", gotStatusCode)
	}
	actualBody = strings.Replace(gotBodyString, workDir, subDir, -1)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("expected body %v, got %v\ndiff:%v", expectBody, actualBody, diff)
	}

	// Status with no history, but FSI working directory has contents
	body["fsi"] = "true"
	gotStatusCode, gotBodyString = JSONAPICallWithBody(http.MethodPost, "/status", body, statusHandler, nil)
	if gotStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", gotStatusCode)
	}
	actualBody = strings.Replace(gotBodyString, workDir, subDir, -1)
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("api response (-want +got):\n%s", diff)
	}

	historyHandler := lib.NewHTTPRequestHandler(run.Inst, "dataset.activity")

	expectNoHistoryBody := `{
  "meta": {
    "code": 422,
    "error": "repo: no history"
  }
}`

	// History with no history
	p := lib.ActivityParams{Ref: "peer/test_ds"}
	gotStatusCode, gotBodyString = JSONAPICallWithBody(http.MethodPost, qhttp.AEActivity.String(), p, historyHandler, nil)
	if gotStatusCode != 422 {
		t.Errorf("expected status code 422, got %d", gotStatusCode)
	}
	if diff := cmp.Diff(expectNoHistoryBody, gotBodyString); diff != "" {
		t.Errorf("expected body %v, got %v\ndiff:%v", expectNoHistoryBody, gotBodyString, diff)
	}

	p.EnsureFSIExists = true
	// History with no history, still returns ErrNoHistory since this route ignores fsi param
	gotStatusCode, gotBodyString = JSONAPICallWithBody(http.MethodPost, qhttp.AEActivity.String(), p, historyHandler, nil)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inst := newTestInstanceWithProfileFromNode(ctx, node)

	tmpDir := os.TempDir()
	workSubdir := "write_test"
	workDir := filepath.Join(tmpDir, workSubdir)
	// Don't create the work directory, it must not exist for checkout to work. Remove if it
	// already exists.
	_ = os.RemoveAll(workDir)

	// TODO(dustmop): Use a TestRunner here, and have it call SaveDataset instead.

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
	_, err := inst.Dataset().Save(ctx, &saveParams)
	if err != nil {
		t.Fatal(err)
	}

	// Checkout the dataset
	checkoutHandler := func(w http.ResponseWriter, r *http.Request) {
		muxVarsToQueryParamMiddleware(lib.NewHTTPRequestHandler(inst, "fsi.checkout")).ServeHTTP(w, r)
	}
	actualStatusCode, actualBody := JSONAPICallWithBody(
		http.MethodPost,
		"/checkout",
		map[string]string{
			"ref": "peer/write_test",
			"dir": workDir,
		},
		checkoutHandler,
		nil,
	)
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody := `{"meta":{"code":200}}`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	writeHandler := func(w http.ResponseWriter, r *http.Request) {
		muxVarsToQueryParamMiddleware(lib.NewHTTPRequestHandler(inst, "fsi.write")).ServeHTTP(w, r)
	}
	p := lib.FSIWriteParams{
		Ref:     "peer/write_test",
		Dataset: &dataset.Dataset{Meta: &dataset.Meta{Title: "oh hai there"}},
	}
	status, strRes := JSONAPICallWithBody(http.MethodPost, "/fsi/write/me/write_test", p, writeHandler, nil)

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inst := newTestInstanceWithProfileFromNode(ctx, node)

	tmpDir := os.TempDir()
	workSubdir := "fsi_checkout_restore"
	workDir := filepath.Join(tmpDir, workSubdir)
	// Don't create the work directory, it must not exist for checkout to work. Remove if it
	// already exists.
	_ = os.RemoveAll(workDir)

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
	res, err := inst.Dataset().Save(ctx, &saveParams)
	if err != nil {
		t.Fatal(err)
	}

	// Save the version from reference for later.
	ref1Version := res.Path

	// Save version 2 with a different title
	saveParams = lib.SaveParams{
		Ref: "me/fsi_checkout_restore",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "title two",
			},
		},
	}
	res, err = inst.Dataset().Save(ctx, &saveParams)
	if err != nil {
		t.Fatal(err)
	}

	checkoutHandler := func(w http.ResponseWriter, r *http.Request) {
		lib.NewHTTPRequestHandler(inst, "fsi.checkout").ServeHTTP(w, r)
	}

	// Checkout the dataset
	actualStatusCode, actualBody := JSONAPICallWithBody(
		http.MethodPost,
		"/checkout",
		map[string]string{
			"ref": "me/fsi_checkout_restore",
			"dir": workDir,
		},
		checkoutHandler,
		nil,
	)
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody := `{"meta":{"code":200}}`
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

	statusHandler := func(w http.ResponseWriter, r *http.Request) {
		lib.NewHTTPRequestHandler(inst, "fsi.status").ServeHTTP(w, r)
	}

	// Status should show that meta is modified
	body := map[string]string{"ref": "peer/fsi_checkout_restore", "fsi": "true"}
	actualStatusCode, actualBody = JSONAPICallWithBody(http.MethodPost, "/status/peer/fsi_checkout_restore?fsi=true", body, statusHandler, nil)
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
	restoreHandler := func(w http.ResponseWriter, r *http.Request) {
		lib.NewHTTPRequestHandler(inst, "fsi.restore").ServeHTTP(w, r)
	}
	actualStatusCode, actualBody = JSONAPICallWithBody(
		http.MethodPost,
		"/restore",
		map[string]string{
			"ref":      "me/fsi_checkout_restore",
			"selector": "meta",
		},
		restoreHandler,
		nil,
	)
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody = `{"meta":{"code":200}}`
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
	actualStatusCode, actualBody = JSONAPICallWithBody(
		http.MethodPost,
		"/restore",
		map[string]string{
			"ref": "me/fsi_checkout_restore",
			// TODO(dlong): Have to pass "dir" to this method. In the test, the ref does
			// not have an FSIPath. Might be because we're using /map/, not sure.
			"dir":     workDir,
			"version": ref1Version,
		},
		restoreHandler,
		nil,
	)
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody = `{"meta":{"code":200}}`
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
func APICall(url string, hf http.HandlerFunc, muxVars map[string]string) (int, string) {
	return APICallWithParams("GET", url, nil, hf, muxVars)
}

// APICallWithParams calls the api and returns the status code and body
func APICallWithParams(method, reqURL string, params map[string]string, hf http.HandlerFunc, muxVars map[string]string) (int, string) {
	// Add parameters from map
	reqParams := url.Values{}
	if params != nil {
		for key := range params {
			reqParams.Set(key, params[key])
		}
	}
	req := httptest.NewRequest(method, reqURL, strings.NewReader(reqParams.Encode()))
	if muxVars != nil {
		req = mux.SetURLVars(req, muxVars)
	}
	setRefStringFromMuxVars(req)
	if err := setMuxVarsToQueryParams(req); err != nil {
		panic(err)
	}
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

func JSONAPICallWithBody(method, reqURL string, data interface{}, hf http.HandlerFunc, muxVars map[string]string) (int, string) {
	enc, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	req := httptest.NewRequest(method, reqURL, bytes.NewReader(enc))
	if muxVars != nil {
		req = mux.SetURLVars(req, muxVars)
	}
	setRefStringFromMuxVars(req)
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
