package api

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDatasetHandlers(t *testing.T) {
	node, teardown := newTestNodeWithNumDatasets(t, 2)
	defer teardown()

	s := newMockDataServer(t)
	defer s.Close()

	inst := newTestInstanceWithProfileFromNode(node)
	h := NewDatasetHandlers(inst, false)

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
		{"GET", "/me/family_relationships/at/map/QmPRjfgUFrH1GxBqujJ3sEvwV3gzHdux1j4g8SLyjbhwot", nil},
		{"GET", "/at/map/QmPRjfgUFrH1GxBqujJ3sEvwV3gzHdux1j4g8SLyjbhwot", nil},
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
