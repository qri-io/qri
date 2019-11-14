package fsi

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func copyDir(sourceDir, destDir string) error {
	entries, err := ioutil.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		sPath := filepath.Join(sourceDir, ent.Name())
		dPath := filepath.Join(destDir, ent.Name())
		data, err := ioutil.ReadFile(sPath)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(dPath, data, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestStatusValid(t *testing.T) {
	ctx := context.Background()
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, _, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	_ = copyDir("testdata/valid_mappings/all_json_components/", paths.firstDir)
	changes, err := fsi.Status(ctx, paths.firstDir)
	if err != nil {
		t.Fatalf(err.Error())
	}
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].SourceFile < changes[j].SourceFile
	})
	actual := ""
	for _, ch := range changes {
		actual += strings.Replace(fmt.Sprintf("%s", ch), paths.firstDir, ".", 1)
	}
	// Construct the expected response by getting the real timestamp from each component.
	expectList := []string{"body", "commit", "meta", "readme", "structure"}
	expect := ""
	for _, cmpName := range expectList {
		var componentFile string
		if cmpName == "body" {
			componentFile = "./body.csv"
		} else if cmpName == "readme" {
			componentFile = "./readme.md"
		} else {
			componentFile = fmt.Sprintf("./%s.json", cmpName)
		}
		st, _ := os.Stat(filepath.Join(paths.firstDir, componentFile))
		mtimeText := st.ModTime().In(time.Local)
		expect = fmt.Sprintf("%s{%s %s add  %s}", expect, componentFile, cmpName, mtimeText)
	}
	if actual != expect {
		t.Errorf("status error didn't match, actual: %s, expect: %s", actual, expect)
	}
}

func TestStatusInvalidDataset(t *testing.T) {
	ctx := context.Background()
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, _, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	_ = copyDir("testdata/invalid_mappings/double_format/", paths.firstDir)
	changes, err := fsi.Status(ctx, paths.firstDir)
	if err != nil {
		t.Fatal(err)
	}

	message := fmt.Sprintf("%s", changes)
	expect := "dataset conflict"
	if !strings.Contains(message, expect) {
		t.Errorf("status error didn't match, expected to contain: %s, got: %s", expect, message)
	}
}

func TestStatusInvalidMeta(t *testing.T) {
	ctx := context.Background()
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, _, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	_ = copyDir("testdata/invalid_mappings/two_metas/", paths.firstDir)
	changes, err := fsi.Status(ctx, paths.firstDir)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !strings.Contains(fmt.Sprintf("%s", changes), "meta conflict") {
		t.Errorf("status should have message about meta conflict")
	}
}

func TestStatusNotFound(t *testing.T) {
	ctx := context.Background()
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, _, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	_ = copyDir("testdata/invalid_mappings/not_found/", paths.firstDir)
	_, err = fsi.Status(ctx, paths.firstDir)
	if err == nil {
		t.Fatalf("expected error, did not get one")
	}
	expect := `no dataset files provided`
	if err.Error() != expect {
		t.Errorf("status error didn't match, actual: %s, expect: %s", err.Error(), expect)
	}
}
