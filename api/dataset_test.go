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

	initCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/", mustFile(t, "testdata/newRequestFromURL.json")},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "init", h.InitHandler, initCases)

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
		{"POST", "/new",
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
		{"POST", "/new",
			map[string]string{
				"body": "testdata/cities/data.csv",
				"file": "testdata/cities/init_dataset.json",
			},
			map[string]string{
				"peername": "peer",
				"name":     "cities",
			},
		},
	}
	runMimeMultipartHandlerTestCases(t, "new mime/multipart", h.InitHandler, newMimeCases)
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
