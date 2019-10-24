package dstest

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/qri-io/dataset"
)

func TestDatasetChecksum(t *testing.T) {
	expect := "085e607818aae2920e0e4b57c321c3b58e17b85d"
	sum := DatasetChecksum(&dataset.Dataset{})
	if sum != expect {
		t.Errorf("empty pod hash mismatch. expected: %s, got: %s", expect, sum)
	}
}

func TestLoadTestCases(t *testing.T) {
	tcs, err := LoadTestCases("testdata")
	if err != nil {
		t.Error(err)
	}
	if len(tcs) == 0 {
		t.Errorf("expected at least one test case to load")
	}
}

func TestBodyFilepath(t *testing.T) {
	fp, err := BodyFilepath("testdata/complete")
	if err != nil {
		t.Error(err.Error())
		return
	}
	if fp != "testdata/complete/body.csv" {
		t.Errorf("%s != %s", "testdata/complete/body.csv", fp)
	}
}

func TestReadInputTransformScript(t *testing.T) {
	if _, _, err := ReadInputTransformScript("bad_dir"); err != os.ErrNotExist {
		t.Error("expected os.ErrNotExist on bad tf script read")
	}
}

func TestNewTestCaseFromDir(t *testing.T) {
	var err error
	if _, err = NewTestCaseFromDir("testdata"); err == nil {
		t.Errorf("expected error")
		return
	}

	tc, err := NewTestCaseFromDir("testdata/complete")
	if err != nil {
		t.Errorf("error reading test dir: %s", err.Error())
		return
	}

	name := "complete"
	if tc.Name != name {
		t.Errorf("expected name to equal: %s. got: %s", name, tc.Name)
	}

	fn := "body.csv"
	if tc.BodyFilename != fn {
		t.Errorf("expected BodyFilename to equal: %s. got: %s", fn, tc.BodyFilename)
	}

	data := []byte(`city,pop,avg_age,in_usa
toronto,40000000,55.5,false
new york,8500000,44.4,true
chicago,300000,44.4,true
chatham,35000,65.25,true
raleigh,250000,50.65,true
`)
	if !bytes.Equal(tc.Body, data) {
		t.Errorf("data mismatch")
	}

	mf := tc.BodyFile()
	if mf.FileName() != tc.BodyFilename {
		t.Errorf("filename mismatch: %s != %s", mf.FileName(), tc.BodyFilename)
	}

	if ts, ok := tc.TransformScriptFile(); !ok {
		t.Errorf("expected tranform script to load")
	} else {
		if ts.FileName() != "transform.star" {
			t.Errorf("expected TransformScript filename to be transform.star")
		}
	}
	tc.TransformScript = nil
	if _, ok := tc.TransformScriptFile(); ok {
		t.Error("shouldn't generate TransformScript File if bytes are nil")
	}

	if vz, ok := tc.VizScriptFile(); !ok {
		t.Errorf("expected viz script to load")
	} else {
		if vz.FileName() != "template.html" {
			t.Errorf("expected VizScript filename to be template.html")
		}
	}
	tc.VizScript = nil
	if _, ok := tc.VizScriptFile(); ok {
		t.Error("shouldn't generate VizScript File if bytes are nil")
	}

	mfdata, err := ioutil.ReadAll(mf)
	if err != nil {
		t.Errorf("error reading file: %s", err.Error())
	}

	if !bytes.Equal(mfdata, data) {
		t.Errorf("memfile data mismatch")
	}

	rendered, err := tc.RenderedFile()
	if err != nil {
		t.Errorf("reading %s: %s", RenderedFilename, err)
	}
	if rendered == nil {
		t.Error("expected rendered to not equal nil")
	}
}
