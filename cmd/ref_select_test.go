package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"testing"
)

func TestRefSelect(t *testing.T) {
	f, err := NewTestFactory(nil)
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
	refs, err := GetCurrentRefSelect(f, []string{"me/explicit_ds"}, -1)
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
	refs, err = GetCurrentRefSelect(f, []string{}, -1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if refs.Ref() != "me/linked_ds" {
		t.Errorf("error: ref_select, actual: %s, expect: %s", refs.Ref(), "me/linked_ds")
	}

	// Explicit has higher precedence than linked
	refs, err = GetCurrentRefSelect(f, []string{"me/explicit_ds"}, -1)
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
	refs, err = GetCurrentRefSelect(f, []string{}, -1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if refs.Ref() != "me/linked_ds" {
		t.Errorf("error: ref_select, actual: %s, expect: %s", refs.Ref(), "me/linked_ds")
	}

	// Remove link
	os.Remove(filepath.Join(workPath, ".qri-ref"))

	// Use dataset
	refs, err = GetCurrentRefSelect(f, []string{}, -1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if refs.Ref() != "me/use_ds" {
		t.Errorf("error: ref_select, actual: %s, expect: %s", refs.Ref(), "me/use_ds")
	}

	// Explicit has higher precedence than use dataset
	refs, err = GetCurrentRefSelect(f, []string{"me/explicit_ds"}, -1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if refs.Ref() != "me/explicit_ds" {
		t.Errorf("error: ref_select, actual: %s, expect: %s", refs.Ref(), "me/explicit_ds")
	}

}
