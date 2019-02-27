package api

import (
	"io/ioutil"
	"net/http/httptest"
	"strings"

	"testing"
)

func TestRegistryHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	h := NewRegistryHandlers(node)

	registryCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/registry/me/counter", nil},
		{"DELETE", "/registry/me/counter", nil},
		{"PATCH", "/", nil},
	}
	runHandlerTestCases(t, "registry", h.RegistryHandler, registryCases)

	registryDatasetsCases := []handlerTestCase{
		{"GET", "/registry/datasets", nil},
	}
	runHandlerTestCases(t, "registryDatasets", h.RegistryDatasetsHandler, registryDatasetsCases)
}

func TestRegistryGet(t *testing.T) {
	node, teardown := newTestNodeWithNumDatasets(t, 1)
	defer teardown()
	h := NewRegistryHandlers(node)

	req := httptest.NewRequest("GET", "/registry/me/ds_0", strings.NewReader(""))
	w := httptest.NewRecorder()
	h.RegistryDatasetHandler(w, req)
	res := w.Result()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	got := string(body)

	expect := `{"data":{"peername":"peer","name":"ds_0","path":"QmAbC0","dataset":{"name":"ds_0","path":"QmAbC0","peername":"peer","qri":"ds:0"},"published":true},"meta":{"code":200}}`
	if got != expect {
		t.Errorf("did not match, got:\n%s\nexpect:\n%s\n", got, expect)
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	node, teardown := newTestNodeWithNumDatasets(t, 1)
	defer teardown()
	h := NewRegistryHandlers(node)

	req := httptest.NewRequest("GET", "/registry/me/not_found", strings.NewReader(""))
	w := httptest.NewRecorder()
	h.RegistryDatasetHandler(w, req)
	res := w.Result()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	got := string(body)

	expect := `{ "meta": { "code": 404, "status": "not found" }, "data": null }`
	if got != expect {
		t.Errorf("did not match, got:\n%s\nexpect:\n%s\n", got, expect)
	}
}
