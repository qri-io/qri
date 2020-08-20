package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo/profile"
	testrepo "github.com/qri-io/qri/repo/test"
)

type testRunner struct {
	Ctx      context.Context
	Profile  *profile.Profile
	Instance *Instance
	Pwd      string
	TmpDir   string
	WorkDir  string
	dsfsTs   func() time.Time
	bookTs   func() int64
}

func newTestRunner(t *testing.T) *testRunner {
	ctx := context.Background()
	dsfsCounter := 0
	dsfsTsFunc := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time {
		dsfsCounter++
		return time.Date(2001, 01, 01, 01, dsfsCounter, 01, 01, time.UTC)
	}

	bookCounter := 0
	bookTsFunc := logbook.NewTimestamp
	logbook.NewTimestamp = func() int64 {
		bookCounter++
		return time.Date(2001, 01, 01, 01, bookCounter, 01, 01, time.UTC).Unix()
	}

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	bus := event.NewBus(ctx)

	// A temporary directory for doing filesystem work.
	tmpDir, err := ioutil.TempDir("", "lib_test_runner")
	if err != nil {
		t.Fatal(err)
	}

	mr, err := testrepo.NewEmptyTestRepo(bus)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), bus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	return &testRunner{
		Ctx: ctx,
		// TODO (b5) - move test profile creation into testRunner constructor
		Profile:  testPeerProfile,
		Instance: NewInstanceFromConfigAndNodeAndBus(ctx, config.DefaultConfigForTesting(), node, bus),
		TmpDir:   tmpDir,
		Pwd:      pwd,
		dsfsTs:   dsfsTsFunc,
		bookTs:   bookTsFunc,
	}
}

func (tr *testRunner) Delete() {
	dsfs.Timestamp = tr.dsfsTs
	logbook.NewTimestamp = tr.bookTs
	os.Chdir(tr.Pwd)
	os.RemoveAll(tr.TmpDir)
}

func (tr *testRunner) MustReadFile(t *testing.T, filename string) string {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func (tr *testRunner) MustWriteTmpFile(t *testing.T, filename, data string) string {
	path := filepath.Join(tr.TmpDir, filename)
	tr.MustWriteFile(t, path, data)
	return path
}

func (tr *testRunner) MustWriteFile(t *testing.T, filename, data string) {
	if err := ioutil.WriteFile(filename, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}

func (tr *testRunner) MakeTmpFilename(filename string) (path string) {
	return filepath.Join(tr.TmpDir, filename)
}

func (tr *testRunner) NiceifyTempDirs(text string) string {
	// Replace the temporary directory
	text = strings.Replace(text, tr.TmpDir, "/tmp", -1)
	// Replace that same directory with symlinks resolved
	realTmp, err := filepath.EvalSymlinks(tr.TmpDir)
	if err == nil {
		text = strings.Replace(text, realTmp, "/tmp", -1)
	}
	return text
}

func (tr *testRunner) ChdirToRoot() {
	os.Chdir(tr.TmpDir)
}

func (tr *testRunner) CreateAndChdirToWorkDir(subdir string) string {
	tr.WorkDir = filepath.Join(tr.TmpDir, subdir)
	err := os.Mkdir(tr.WorkDir, 0755)
	if err != nil {
		panic(err)
	}
	err = os.Chdir(tr.WorkDir)
	if err != nil {
		panic(err)
	}
	return tr.WorkDir
}

func (tr *testRunner) MustSaveFromBody(t *testing.T, dsName, bodyFilename string) *dataset.Dataset {
	if !dsref.IsValidName(dsName) {
		t.Fatalf("invalid dataset name: %q", dsName)
	}
	m := NewDatasetMethods(tr.Instance)
	p := SaveParams{
		Ref:      fmt.Sprintf("peer/%s", dsName),
		BodyPath: bodyFilename,
	}
	res := &dataset.Dataset{}
	if err := m.Save(&p, res); err != nil {
		t.Fatal(err)
	}
	return res
}

func (tr *testRunner) SaveWithParams(p *SaveParams) (dsref.Ref, error) {
	m := NewDatasetMethods(tr.Instance)
	res := &dataset.Dataset{}
	if err := m.Save(p, res); err != nil {
		return dsref.Ref{}, err
	}
	return dsref.ConvertDatasetToVersionInfo(res).SimpleRef(), nil
}

func (tr *testRunner) Diff(left, right, selector string) (string, error) {
	m := NewDatasetMethods(tr.Instance)
	p := DiffParams{
		LeftSide:           left,
		RightSide:          right,
		UseLeftPrevVersion: right == "",
		Selector:           selector,
	}
	r := DiffResponse{}
	err := m.Diff(&p, &r)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (tr *testRunner) DiffWithParams(p *DiffParams) (string, error) {
	m := NewDatasetMethods(tr.Instance)
	r := DiffResponse{}
	err := m.Diff(p, &r)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (tr *testRunner) Init(refstr, format string) error {
	ref, err := dsref.Parse(refstr)
	if err != nil {
		return err
	}
	m := NewFSIMethods(tr.Instance)
	out := ""
	p := InitFSIDatasetParams{
		Name:   ref.Name,
		Dir:    tr.WorkDir,
		Format: format,
	}
	return m.InitDataset(&p, &out)
}

func (tr *testRunner) Checkout(refstr, dir string) error {
	m := NewFSIMethods(tr.Instance)
	out := ""
	p := CheckoutParams{
		Ref: refstr,
		Dir: dir,
	}
	return m.Checkout(&p, &out)
}
