package api

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
)

func TestUpdateHandlers(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "update_handlers")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfigForTesting()
	cfg.Store.Type = "map"
	cfg.Repo.Type = "mem"

	inst, err := lib.NewInstance(tmpDir,
		lib.OptConfig(cfg),
	)
	if err != nil {
		t.Fatal(err)
	}

	// node, teardown := newTestNode(t)
	// defer teardown()
	// s := newMockDataServer(t)
	// defer s.Close()

	// inst := lib.NewInstanceFromConfigAndNode(, node)
	h := UpdateHandlers{UpdateMethods: lib.NewUpdateMethods(inst), ReadOnly: false}

	listCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", nil},
		// {"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "list", h.UpdatesHandler, listCases, true)

	logCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
	}
	runHandlerTestCases(t, "update log", h.LogsHandler, logCases, false)

	runUpdateCases := []handlerMimeMultipartTestCase{
		{"OPTIONS", "/update/run", nil, nil},
		{"GET", "/update/run", nil, nil},
		{"POST", "/update/run/me/cities", nil, map[string]string{
			"secrets": "bad request",
		}},
		{"POST", "/update/run/me/cities", nil, map[string]string{
			"secrets": `{"key":"value"}`,
		}},
	}
	runMimeMultipartHandlerTestCases(t, "update run", h.RunHandler, runUpdateCases)

	serviceCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
	}
	runHandlerTestCases(t, "update service", h.ServiceHandler, serviceCases, false)
}
