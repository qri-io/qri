package cmd

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qri-io/qri/dsref"
)

func TestBasicRefSelect(t *testing.T) {
	refs := NewEmptyRefSelect()
	if refs.Ref() != "" {
		t.Errorf("expected ref \"\", got %s", refs.Ref())
	}
	if strings.Join(refs.RefList(), ",") != "" {
		t.Errorf("expected ref list \"\", got %s", refs.RefList())
	}
	if refs.String() != "" {
		t.Errorf("expected ref string \"\", got %s", refs.String())
	}
	if refs.Dir() != "" {
		t.Errorf("expected ref dir \"\", got %s", refs.Dir())
	}
	if !refs.IsExplicit() {
		t.Errorf("expected ref isExplict true, got %t", refs.IsExplicit())
	}
	if refs.IsLinked() {
		t.Errorf("expected ref isLinked false, got %t", refs.IsLinked())
	}

	refs = NewExplicitRefSelect("peername/test_ds")
	if refs.Ref() != "peername/test_ds" {
		t.Errorf("expected ref \"peername/test_ds\", got %s", refs.Ref())
	}
	if strings.Join(refs.RefList(), ",") != "peername/test_ds" {
		t.Errorf("expected ref list \"peername/test_ds\", got %s", refs.RefList())
	}
	if refs.String() != "" {
		t.Errorf("expected ref string \"\", got %s", refs.String())
	}
	if refs.Dir() != "" {
		t.Errorf("expected ref dir \"\", got %s", refs.Dir())
	}
	if !refs.IsExplicit() {
		t.Errorf("expected ref isExplict true, got %t", refs.IsExplicit())
	}
	if refs.IsLinked() {
		t.Errorf("expected ref isLinked false, got %t", refs.IsLinked())
	}

	refs = NewListOfRefSelects([]string{"peername/test_ds", "peername/another_ds"})
	if refs.Ref() != "peername/test_ds" {
		t.Errorf("expected ref \"peername/test_ds\", got %s", refs.Ref())
	}
	if strings.Join(refs.RefList(), ",") != "peername/test_ds,peername/another_ds" {
		t.Errorf("expected ref list \"peername/test_ds,peername/another_ds\", got %s", refs.RefList())
	}
	if refs.String() != "" {
		t.Errorf("expected ref string \"\", got %s", refs.String())
	}
	if refs.Dir() != "" {
		t.Errorf("expected ref dir \"\", got %s", refs.Dir())
	}
	if !refs.IsExplicit() {
		t.Errorf("expected ref isExplict true, got %t", refs.IsExplicit())
	}
	if refs.IsLinked() {
		t.Errorf("expected ref isLinked false, got %t", refs.IsLinked())
	}

	refs = NewLinkedDirectoryRefSelect(dsref.MustParse("peername/test_ds"), "path/to/test_ds")
	if refs.Ref() != "peername/test_ds" {
		t.Errorf("expected ref \"peername/test_ds\", got %s", refs.Ref())
	}
	if strings.Join(refs.RefList(), ",") != "peername/test_ds" {
		t.Errorf("expected ref list \"peername/test_ds\", got %s", refs.RefList())
	}
	if refs.String() != "for linked dataset [peername/test_ds]" {
		t.Errorf("expected ref string \"for linked dataset [peername/test_ds]\", got %s", refs.String())
	}
	if refs.Dir() != "path/to/test_ds" {
		t.Errorf("expected ref dir \"path/to/test_ds\", got %s", refs.Dir())
	}
	if refs.IsExplicit() {
		t.Errorf("expected ref isExplict false, got %t", refs.IsExplicit())
	}
	if !refs.IsLinked() {
		t.Errorf("expected ref isLinked true, got %t", refs.IsLinked())
	}

	refs = NewUsingRefSelect("peername/test_ds")
	if refs.Ref() != "peername/test_ds" {
		t.Errorf("expected ref \"peername/test_ds\", got %s", refs.Ref())
	}
	if strings.Join(refs.RefList(), ",") != "peername/test_ds" {
		t.Errorf("expected ref list \"peername/test_ds\", got %s", refs.RefList())
	}
	if refs.String() != "using dataset [peername/test_ds]" {
		t.Errorf("expected ref string \"using dataset [peername/test_ds]\", got %s", refs.String())
	}
	if refs.Dir() != "" {
		t.Errorf("expected ref dir \"\", got %s", refs.Dir())
	}
	if refs.IsExplicit() {
		t.Errorf("expected ref isExplict false, got %t", refs.IsExplicit())
	}
	if refs.IsLinked() {
		t.Errorf("expected ref isLinked false, got %t", refs.IsLinked())
	}
}

func TestGetCurrentRefSelect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Fatalf(err.Error())
	}

	rootPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	qriPath := filepath.Join(rootPath, "qri")
	err = os.MkdirAll(qriPath, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	f.qriRepoPath = qriPath
	workPath := filepath.Join(rootPath, "work")
	err = os.MkdirAll(workPath, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(qriPath)
	defer os.RemoveAll(workPath)

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	os.Chdir(workPath)
	defer os.Chdir(pwd)

	// Explicit argument
	refs, err := GetCurrentRefSelect(f, []string{"me/explicit_ds"}, -1, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if refs.Ref() != "me/explicit_ds" {
		t.Errorf("error: ref_select, actual: %s, expect: %s", refs.Ref(), "me/explicit_ds")
	}

	// Linked directory
	data := []byte("me/linked_ds")
	err = ioutil.WriteFile(filepath.Join(workPath, ".qri-ref"), data, os.ModePerm)
	if err != nil {
		t.Fatalf(err.Error())
	}
	refs, err = GetCurrentRefSelect(f, []string{}, -1, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if refs.Ref() != "me/linked_ds" {
		t.Errorf("error: ref_select, actual: %s, expect: %s", refs.Ref(), "me/linked_ds")
	}

	// Explicit has higher precedence than linked
	refs, err = GetCurrentRefSelect(f, []string{"me/explicit_ds"}, -1, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if refs.Ref() != "me/explicit_ds" {
		t.Errorf("error: ref_select, actual: %s, expect: %s", refs.Ref(), "me/explicit_ds")
	}

	// Use dataset, linked has high precedence
	data = []byte("[{\"peername\": \"me\",\"name\":\"use_ds\"}]")
	err = ioutil.WriteFile(filepath.Join(qriPath, "selected_refs.json"), data, os.ModePerm)
	if err != nil {
		t.Fatalf(err.Error())
	}
	refs, err = GetCurrentRefSelect(f, []string{}, -1, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if refs.Ref() != "me/linked_ds" {
		t.Errorf("error: ref_select, actual: %s, expect: %s", refs.Ref(), "me/linked_ds")
	}

	// Remove link
	os.Remove(filepath.Join(workPath, ".qri-ref"))

	// Use dataset
	refs, err = GetCurrentRefSelect(f, []string{}, -1, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if refs.Ref() != "me/use_ds" {
		t.Errorf("error: ref_select, actual: %s, expect: %s", refs.Ref(), "me/use_ds")
	}

	// Explicit has higher precedence than use dataset
	refs, err = GetCurrentRefSelect(f, []string{"me/explicit_ds"}, -1, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if refs.Ref() != "me/explicit_ds" {
		t.Errorf("error: ref_select, actual: %s, expect: %s", refs.Ref(), "me/explicit_ds")
	}

}

func TestGetCurrentRefSelectUsingTwoArgs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// If two are allowed, refs be have length 2
	refs, err := GetCurrentRefSelect(f, []string{"me/first_ds", "me/second_ds"}, 2, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(refs.RefList()) != 2 {
		t.Fatalf("error: ref_select.len, actual: %d, expect: %d", len(refs.RefList()), 2)
	}
	if refs.RefList()[0] != "me/first_ds" {
		t.Fatalf("error: ref[0], actual: %s, expect: %s", refs.RefList()[0], "me/first_ds")
	}
	if refs.RefList()[1] != "me/second_ds" {
		t.Fatalf("error: ref[0], actual: %s, expect: %s", refs.RefList()[1], "me/second_ds")
	}
}
