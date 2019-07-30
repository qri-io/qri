package api

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
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
