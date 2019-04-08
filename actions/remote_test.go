package actions

import (
	"net/http"
	"net/http/httptest"
	"testing"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/dag"
	"github.com/qri-io/qri/repo"
)

func TestDsyncStartPush(t *testing.T) {
	node := newTestNode(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, h *http.Request) {
		result := ReceiveResult{
			Success:   true,
			SessionID: "abc123def",
		}
		util.WriteResponse(w, result)
	}))
	defer server.Close()

	dagInfo := dag.Info{}
	ref := repo.DatasetRef{Peername: "me"}
	sessionID, _, err := DsyncStartPush(node, &dagInfo, server.URL, &ref)
	if err != nil {
		t.Fatal(err.Error())
	}

	expect := "abc123def"
	if sessionID != expect {
		t.Errorf("error sessionID expected: \"%s\", got: \"%s\"", expect, sessionID)
	}
}

func TestDsyncCompletePush(t *testing.T) {
	node := newTestNode(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, h *http.Request) {
		util.WriteResponse(w, "Success")
	}))
	defer server.Close()

	sessionID := "abc123def"
	err := DsyncCompletePush(node, server.URL, sessionID)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
}
