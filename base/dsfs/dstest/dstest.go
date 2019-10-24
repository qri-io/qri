// Package dstest defines an interface for reading test cases from static files
// leveraging directories of test dataset input files & expected output files
package dstest

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	logger "github.com/ipfs/go-log"
	"github.com/jinzhu/copier"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/ugorji/go/codec"
)

var log = logger.Logger("dstest")

const (
	// InputDatasetFilename is the filename to use for an input dataset
	InputDatasetFilename = "input.dataset.json"
	// ExpectDatasetFilename is the filename to use to compare expected outputs
	ExpectDatasetFilename = "expect.dataset.json"
	// RenderedFilename is the file that represents an executed viz script
	RenderedFilename = "rendered.html"
)

// TestCase is a dataset test case, usually built from a
// directory of files for use in tests.
// All files are optional for TestCase, but may be required
// by the test itself.
type TestCase struct {
	// Path to the director on the local filesystem this test case is loaded from
	Path string
	// Name is the casename, should match directory name
	Name string
	// body.csv,body.json, etc
	BodyFilename string
	// test body in expected data format
	Body []byte
	// Filename of Transform Script
	TransformScriptFilename string
	// TransformScript bytes if one exists
	TransformScript []byte
	// Filename of Viz Script
	VizScriptFilename string
	// VizScript bytes if one exists
	VizScript []byte
	// Input is intended file for test input
	// loads from input.dataset.json
	Input *dataset.Dataset
	//  Expect should match test output
	// loads from expect.dataset.json
	Expect *dataset.Dataset
}

// DatasetChecksum generates a fast, insecure hash of an encoded dataset,
// useful for checking that expected dataset values haven't changed
func DatasetChecksum(ds *dataset.Dataset) string {
	buf := &bytes.Buffer{}
	h := &codec.CborHandle{}
	h.Canonical = true
	if err := codec.NewEncoder(buf, h).Encode(ds); err != nil {
		panic(err)
	}

	sum := sha1.Sum(buf.Bytes())
	return hex.EncodeToString(sum[:])
}

var testCaseCache = make(map[string]TestCase)

// BodyFile creates a new in-memory file from data & filename properties
func (t TestCase) BodyFile() qfs.File {
	return qfs.NewMemfileBytes(t.BodyFilename, t.Body)
}

// TransformScriptFile creates a qfs.File from testCase transform script data
func (t TestCase) TransformScriptFile() (qfs.File, bool) {
	if t.TransformScript == nil {
		return nil, false
	}
	return qfs.NewMemfileBytes(t.TransformScriptFilename, t.TransformScript), true
}

// VizScriptFile creates a qfs.File from testCase transform script data
func (t TestCase) VizScriptFile() (qfs.File, bool) {
	if t.VizScript == nil {
		return nil, false
	}
	return qfs.NewMemfileBytes(t.VizScriptFilename, t.VizScript), true
}

// RenderedFile returns a qfs.File of the rendered file if one exists
func (t TestCase) RenderedFile() (qfs.File, error) {
	path := filepath.Join(t.Path, RenderedFilename)
	f, err := os.Open(path)
	return qfs.NewMemfileReader(RenderedFilename, f), err
}

// BodyFilepath retuns the path to the first valid data file it can find,
// which is a file named "data" that ends in an extension we support
func BodyFilepath(dir string) (string, error) {
	for _, df := range dataset.SupportedDataFormats() {
		path := fmt.Sprintf("%s/body.%s", dir, df)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return path, nil
		}
	}
	return "", os.ErrNotExist
}

// LoadTestCases loads a directory of case directories
func LoadTestCases(dir string) (tcs map[string]TestCase, err error) {
	tcs = map[string]TestCase{}
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}
	for _, fi := range fis {
		if fi.IsDir() {
			if tc, err := NewTestCaseFromDir(filepath.Join(dir, fi.Name())); err == nil {
				tcs[fi.Name()] = tc
			}
		}
	}
	return
}

// NewTestCaseFromDir creates a test case from a directory of static test files
// dir should be the path to the directory to check, and any parsing errors will
// be logged using t.Log methods
func NewTestCaseFromDir(dir string) (tc TestCase, err error) {
	// TODO (b5): for now we need to disable the cache b/c copier.Copy can't
	// copy unexported fields, which includes script files. We should switch
	// the testcase.Input field to a method that creates new datset instances
	// with fresh files on each call of input, this'll let us restore cache use
	// and speed tests up again
	// if got, ok := testCaseCache[dir]; ok {
	// 	tc = TestCase{}
	// 	copier.Copy(&tc, &got)
	// 	return
	// }
	foundTestData := false

	tc = TestCase{
		Path: dir,
		Name: filepath.Base(dir),
	}

	tc.Input, err = ReadDataset(dir, InputDatasetFilename)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		} else {
			return tc, fmt.Errorf("%s reading input dataset: %s", tc.Name, err.Error())
		}
	} else {
		foundTestData = true
	}
	if tc.Input == nil {
		tc.Input = &dataset.Dataset{}
	}
	tc.Input.Name = tc.Name

	if tc.Body, tc.BodyFilename, err = ReadBodyData(dir); err != nil {
		if err == os.ErrNotExist {
			// Body is optional
			err = nil
		} else {
			return tc, fmt.Errorf("error reading test case body for dir: %s: %s", dir, err.Error())
		}
	} else {
		foundTestData = true
		tc.Input.SetBodyFile(qfs.NewMemfileBytes(tc.BodyFilename, tc.Body))
	}

	if tc.TransformScript, tc.TransformScriptFilename, err = ReadInputTransformScript(dir); err != nil {
		if err == os.ErrNotExist {
			// TransformScript is optional
			err = nil
		} else {
			return tc, fmt.Errorf("reading transform script: %s", err.Error())
		}
	} else {
		foundTestData = true
		if tc.Input.Transform == nil {
			tc.Input.Transform = &dataset.Transform{}
		}
		tc.Input.Transform.SetScriptFile(qfs.NewMemfileBytes(tc.TransformScriptFilename, tc.TransformScript))
	}

	if tc.VizScript, tc.VizScriptFilename, err = ReadInputVizScript(dir); err != nil {
		if err == os.ErrNotExist {
			// VizScript is optional
			err = nil
		} else {
			return tc, fmt.Errorf("reading viz script: %s", err.Error())
		}
	} else {
		foundTestData = true
		if tc.Input.Viz == nil {
			tc.Input.Viz = &dataset.Viz{}
		}
		tc.Input.Viz.SetScriptFile(qfs.NewMemfileBytes(tc.VizScriptFilename, tc.VizScript))
	}

	tc.Expect, err = ReadDataset(dir, ExpectDatasetFilename)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		} else {
			return tc, fmt.Errorf("%s: error loading expect dataset: %s", tc.Name, err)
		}
	} else {
		foundTestData = true
	}

	if !foundTestData {
		return tc, fmt.Errorf("%s no test data at path: %s", tc.Name, dir)
	}

	preserve := TestCase{}
	copier.Copy(&preserve, &tc)
	testCaseCache[dir] = preserve
	return
}

// ReadDataset grabs a dataset for a given dir for a given filename
func ReadDataset(dir, filename string) (*dataset.Dataset, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		log.Info(err.Error())
		return nil, err
	}

	ds := &dataset.Dataset{}
	return ds, json.Unmarshal(data, ds)
}

// ReadBodyData grabs input data
func ReadBodyData(dir string) ([]byte, string, error) {
	for _, df := range dataset.SupportedDataFormats() {
		path := fmt.Sprintf("%s/body.%s", dir, df)
		if f, err := os.Open(path); err == nil {
			data, err := ioutil.ReadAll(f)
			return data, fmt.Sprintf("body.%s", df), err
		}
	}
	return nil, "", os.ErrNotExist
}

// ReadInputTransformScript grabs input transform bytes
func ReadInputTransformScript(dir string) ([]byte, string, error) {
	path := filepath.Join(dir, "transform.star")
	if f, err := os.Open(path); err == nil {
		data, err := ioutil.ReadAll(f)
		return data, "transform.star", err
	}
	return nil, "", os.ErrNotExist
}

// ReadInputVizScript grabs input viz script bytes
func ReadInputVizScript(dir string) ([]byte, string, error) {
	path := filepath.Join(dir, "template.html")
	if f, err := os.Open(path); err == nil {
		data, err := ioutil.ReadAll(f)
		return data, "template.html", err
	}
	return nil, "", os.ErrNotExist
}
