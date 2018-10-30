package api

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDatasetHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	s := newMockDataServer(t)
	defer s.Close()

	h := NewDatasetHandlers(node, false)

	listCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "list", h.ListHandler, listCases)

	// TODO: Remove this case, update API snapshot.
	initCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/", mustFile(t, "testdata/newRequestFromURL.json")},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "init", h.SaveHandler, initCases)

	saveCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/", mustFile(t, "testdata/newRequestFromURL.json")},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "save", h.SaveHandler, saveCases)

	getCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/me/family_relationships", nil},
		{"GET", "/me/family_relationships/at/map/QmPRjfgUFrH1GxBqujJ3sEvwV3gzHdux1j4g8SLyjbhwot", nil},
		{"GET", "/at/map/QmPRjfgUFrH1GxBqujJ3sEvwV3gzHdux1j4g8SLyjbhwot", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "get", h.GetHandler, getCases)

	bodyCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/body/me/family_relationships", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "body", h.BodyHandler, bodyCases)

	renameCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/rename", mustFile(t, "testdata/renameRequest.json")},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "rename", h.RenameHandler, renameCases)

	exportCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/export/me/cities", nil},
		{"GET", "/export/me/cities/at/map/QmPRjfgUFrH1GxBqujJ3sEvwV3gzHdux1j4g8SLyjbhwot", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "export", h.ZipDatasetHandler, exportCases)

	publishCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/publish/", nil},
		{"POST", "/publish/me/cities", nil},
		{"DELETE", "/publish/me/cities", nil},
	}
	runHandlerTestCases(t, "publish", h.PublishHandler, publishCases)

	// TODO: Perhaps add an option to runHandlerTestCases to set Content-Type, then combine
	// `runHandlerZipPostTestCases` with `runHandlerTestCases`.
	unpackCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/unpack/", mustFile(t, "testdata/exported.zip")},
	}
	runHandlerZipPostTestCases(t, "unpack", h.UnpackHandler, unpackCases)

	diffCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", mustFile(t, "testdata/diffRequest.json")},
		{"GET", "/", mustFile(t, "testdata/diffRequestPlusMinusColor.json")},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "diff", h.DiffHandler, diffCases)

	removeCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", nil},
		{"POST", "/remove/me/cities", nil},
		{"POST", "/remove/at/map/QmPRjfgUFrH1GxBqujJ3sEvwV3gzHdux1j4g8SLyjbhwot", nil},
	}
	runHandlerTestCases(t, "remove", h.RemoveHandler, removeCases)

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
