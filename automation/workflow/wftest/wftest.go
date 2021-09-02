package wftest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jinzhu/copier"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/base/dsfs/dstest"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/profile"
)

const (
	// InputWorkflowFilename is the filename to use for an input workflow
	InputWorkflowFilename = "input.workflow.json"
	// ExpectWorkflowFilename is the filename to use to compare expected outputs
	ExpectWorkflowFilename = "expect.workflow.json"
)

var (
	defaultTestCasesDir = ""
	testCaseCache       = []*TestCase{}
)

// TestCase is a workflow test case, usually built from a
// directory of files for use in tests
// Each workflow for each test case must have an associated
// dataset
type TestCase struct {
	// Path to the directory on the local filesystem this test case is loaded from
	Path string
	// Name is the casename, should match the directory name
	Name string
	// Input is intended file for test input
	// loads from input.workflow.json
	Input *workflow.Workflow
	// Expect should match the test output
	// loads from expect.workflow.json
	Expect  *workflow.Workflow
	Dataset *dstest.TestCase
}

// LoadDefaultTestCases loads test cases from this package
// the OwnerID is the first key in the `auth/key/test/keys.go`, which is the
// primary key used in the default test configs
func LoadDefaultTestCases() ([]*TestCase, error) {
	created, err := time.Parse(time.RFC3339, "2021-08-03T10:33:36.23224-04:00")
	if err != nil {
		return nil, err
	}
	return []*TestCase{
		{
			Name: "no_change",
			Input: &workflow.Workflow{
				ID:      "9e45m9ll-b366-0945-2743-8mm90731jl72",
				OwnerID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Created: &created,
				Active:  true,
			},
			Expect: &workflow.Workflow{
				ID:      "9e45m9ll-b366-0945-2743-8mm90731jl72",
				OwnerID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Created: &created,
				Active:  true,
			},
			Dataset: &dstest.TestCase{
				Name: "no_change",
				Input: &dataset.Dataset{
					Transform: &dataset.Transform{
						Steps: []*dataset.TransformStep{
							{
								Name:   "transform",
								Syntax: "starlark",
								Script: "load(\"dataframe.star\", \"dataframe\")\nds = dataset.latest()\nbody = '''a,b,c\n1,2,3\n4,5,6\n'''\nds.body = dataframe.read_csv(body)\ndataset.commit(ds)",
							},
						},
					},
				},
			},
		},
		{
			Name: "now",
			Input: &workflow.Workflow{
				ID:      "1d79b0ff-a133-4731-9892-5ee01842ca81",
				OwnerID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Created: &created,
				Active:  true,
			},
			Expect: &workflow.Workflow{
				ID:      "1d79b0ff-a133-4731-9892-5ee01842ca81",
				OwnerID: "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Created: &created,
				Active:  true,
			},
			Dataset: &dstest.TestCase{
				Name: "now",
				Input: &dataset.Dataset{
					Transform: &dataset.Transform{
						Steps: []*dataset.TransformStep{
							{
								Syntax: "starlark",
								Name:   "setup",
								Script: "load(\"encoding/csv.star\", \"csv\")\nload(\"time.star\", \"time\")\nload(\"dataframe.star\", \"dataframe\")\nds = dataset.latest()",
							},
							{
								Syntax: "starlark",
								Name:   "transform",
								Script: "currentTime = time.now()\nbody = [\n    ['timestamp']\n  ]\n  body.append([str(currentTime)])\ntheCSV = csv.write_all(body)\n\nds.body = dataframe.read_csv(theCSV)\ndataset.commit(ds)",
							},
						},
					},
				},
			},
		},
	}, nil
}

// LoadTestCases loads a directory of case directories
func LoadTestCases(dir string) (tcs []*TestCase, err error) {
	tcs = []*TestCase{}
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}
	for _, fi := range fis {
		if fi.IsDir() {
			tc, err := NewTestCaseFromDir(filepath.Join(dir, fi.Name()))
			if err != nil {
				return nil, err
			}
			tcs = append(tcs, tc)
		}
	}
	return
}

// NewTestCaseFromDir creates a test case from a directory of static test files
// dir should be the path to the directory to check
func NewTestCaseFromDir(dir string) (tc *TestCase, err error) {
	dsTestCase, err := dstest.NewTestCaseFromDir(dir)
	if err != nil {
		return nil, err
	}

	tc = &TestCase{
		Path:    dir,
		Name:    filepath.Base(dir),
		Dataset: &dsTestCase,
	}

	tc.Input, err = ReadWorkflow(filepath.Join(dir, InputWorkflowFilename))
	if err != nil {
		return nil, fmt.Errorf("%s reading input workflow: %s", tc.Name, err)
	}

	tc.Expect, err = ReadWorkflow(filepath.Join(dir, ExpectWorkflowFilename))
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		} else {
			return nil, fmt.Errorf("%s: error loading expect workflow: %s", tc.Name, err)
		}
	}

	preserve := TestCase{}
	copier.Copy(&preserve, &tc)
	testCaseCache = append(testCaseCache, tc)
	return
}

// ReadWorkflow grabs a workflow from a given filepath
func ReadWorkflow(filepath string) (*workflow.Workflow, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	wf := &workflow.Workflow{}
	return wf, json.Unmarshal(data, wf)
}

// A TestRunner give you all the information needed to save workflow
type TestRunner interface {
	Instance() *lib.Instance
	Owner() *profile.Profile
	Context() context.Context
	WorkflowStore() workflow.Store
}

// MustAddWorkflowsFromDir adds workflows and their associated datasets,
// that have been loaded from a directory, to the test runner's instance
// and workflow store
func MustAddWorkflowsFromDir(t *testing.T, tr TestRunner, dir string) {
	tcs, err := LoadTestCases(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := addWorkflowsToTestRunner(tr, tcs); err != nil {
		t.Fatal(err)
	}
}

// MustAddDefaultWorkflows adds the default workflows and their associated
// datasets to the test runner's instance and workflow store
func MustAddDefaultWorkflows(t *testing.T, tr TestRunner) {
	tcs, err := LoadDefaultTestCases()
	if err != nil {
		t.Fatal(err)
	}
	if err := addWorkflowsToTestRunner(tr, tcs); err != nil {
		t.Fatal(err)
	}
}

// addWorkflowToTestRunner adds workflows and their associated datasets
// to the test runner's instance and workflow.Store
func addWorkflowsToTestRunner(tr TestRunner, tcs []*TestCase) error {
	var err error
	owner := tr.Owner()
	inst := tr.Instance()
	wfs := tr.WorkflowStore()
	ctx := tr.Context()

	if owner == nil {
		return fmt.Errorf("missing profile")
	}
	if inst == nil {
		return fmt.Errorf("missing instance")
	}
	if wfs == nil {
		return fmt.Errorf("missing workflow store")
	}

	for _, tc := range tcs {
		ds := tc.Dataset.Input
		wf := tc.Input
		wf.OwnerID = owner.ID
		ref := fmt.Sprintf("%s/%s", owner.Peername, tc.Name)

		// TODO(ramfox): when we can save with out a body or structure, remove the `Apply` flag
		ds, err = inst.Dataset().Save(ctx, &lib.SaveParams{Ref: ref, Dataset: ds, Apply: true})
		if err != nil {
			return fmt.Errorf("saving dataset %q: %w", ref, err)
		}
		wf.InitID = ds.ID
		wf, err = wfs.Put(ctx, wf)
		if err != nil {
			return fmt.Errorf("adding workflow for dataset %q: %w", ref, err)
		}
	}
	return nil
}
