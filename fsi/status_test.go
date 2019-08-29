package fsi

import (
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
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	_ = copyDir("testdata/valid_mappings/all_json_components/", paths.firstDir)
	changes, err := fsi.Status(paths.firstDir)
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
	// Construct the expected repsonse by getting the real timestamp from each component.
	expectList := []string{"commit", "meta", "schema", "structure", "transform", "viz", "body"}
	expect := ""
	for _, cmpName := range expectList {
		var componentFile string
		if cmpName == "body" {
			componentFile = "body.csv"
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
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	_ = copyDir("testdata/invalid_mappings/double_format/", paths.firstDir)
	_, err = fsi.Status(paths.firstDir)
	if err == nil {
		t.Fatalf("expected error, did not get one")
	}
	// TODO(dlong): Kind of annoying, this error message is not deterministic.
	expectOne := `dataset is defined in two places: dataset.json and dataset.yaml. please remove one`
	expectTwo := `dataset is defined in two places: dataset.yaml and dataset.json. please remove one`
	if err.Error() != expectOne && err.Error() != expectTwo {
		t.Errorf("status error didn't match, actual: %s, expect: %s", err.Error(), expectOne)
	}
}

func TestStatusInvalidMeta(t *testing.T) {
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	_ = copyDir("testdata/invalid_mappings/two_metas/", paths.firstDir)
	_, err = fsi.Status(paths.firstDir)
	if err == nil {
		t.Fatalf("expected error, did not get one")
	}
	// TODO(dlong): Kind of annoying, this error message is not deterministic.
	expectOne := `meta is defined in two places: dataset.yaml and meta.json. please remove one`
	expectTwo := `meta is defined in two places: meta.json and dataset.yaml. please remove one`
	if err.Error() != expectOne && err.Error() != expectTwo {
		t.Errorf("status error didn't match, actual: %s, expect: %s", err.Error(), expectOne)
	}
}

func TestStatusNotFound(t *testing.T) {
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	_ = copyDir("testdata/invalid_mappings/not_found/", paths.firstDir)
	_, err = fsi.Status(paths.firstDir)
	if err == nil {
		t.Fatalf("expected error, did not get one")
	}
	expect := `no dataset files provided`
	if err.Error() != expect {
		t.Errorf("status error didn't match, actual: %s, expect: %s", err.Error(), expect)
	}
}
