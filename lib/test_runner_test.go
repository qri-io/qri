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
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/profile"
	remotemock "github.com/qri-io/qri/remote/mock"
)

type testRunner struct {
	Ctx      context.Context
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

	// A temporary directory for doing filesystem work.
	tmpDir, err := ioutil.TempDir("", "lib_test_runner")
	if err != nil {
		t.Fatal(err)
	}

	qriPath := filepath.Join(tmpDir, "qri")
	if err := os.Mkdir(qriPath, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	cfg := testcfg.DefaultMemConfigForTesting()

	// create new instance!
	inst, err := NewInstance(
		ctx,
		// NewInstance requires a qriPath, even if the repo & all stores are in mem
		qriPath,
		OptConfig(cfg),
		// ensure we create a mock registry client for testing
		OptRemoteClientConstructor(remotemock.NewClient),
	)
	if err != nil {
		t.Fatal(err)
	}
	return &testRunner{
		Ctx: ctx,
		// TODO (b5) - move test profile creation into testRunner constructor
		Instance: inst,
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

func (tr *testRunner) MustOwner(t *testing.T) *profile.Profile {
	owner, err := tr.Instance.activeProfile(tr.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	return owner
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
	tr.WorkDir = PathJoinPosix(tr.TmpDir, subdir)
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
	pro, err := tr.Instance.activeProfile(tr.Ctx)
	if err != nil {
		t.Fatalf("error getting active profile: %s", err)
	}
	p := SaveParams{
		Ref:      fmt.Sprintf("%s/%s", pro.Peername, dsName),
		BodyPath: bodyFilename,
	}
	res, err := tr.Instance.Dataset().Save(tr.Ctx, &p)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func (tr *testRunner) SaveWithParams(p *SaveParams) (dsref.Ref, error) {
	res, err := tr.Instance.Dataset().Save(tr.Ctx, p)
	if err != nil {
		return dsref.Ref{}, err
	}
	return dsref.ConvertDatasetToVersionInfo(res).SimpleRef(), nil
}

func (tr *testRunner) MustGet(t *testing.T, ref string) *dataset.Dataset {
	p := GetParams{Ref: ref}
	res, err := tr.Instance.Dataset().Get(tr.Ctx, &p)
	if err != nil {
		t.Fatal(err)
	}
	return res.Value.(*dataset.Dataset)
}

func (tr *testRunner) ApplyWithParams(ctx context.Context, p *ApplyParams) (*dataset.Dataset, error) {
	res, err := tr.Instance.Automation().Apply(ctx, p)
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

func (tr *testRunner) Diff(left, right, selector string) (string, error) {
	p := DiffParams{
		LeftSide:           left,
		RightSide:          right,
		UseLeftPrevVersion: right == "",
		Selector:           selector,
	}
	r, err := tr.Instance.Diff().Diff(tr.Ctx, &p)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(*r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (tr *testRunner) DiffWithParams(p *DiffParams) (string, error) {
	r, err := tr.Instance.Diff().Diff(tr.Ctx, p)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(*r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
