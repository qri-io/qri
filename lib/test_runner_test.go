package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
	testrepo "github.com/qri-io/qri/repo/test"
)

type testRunner struct {
	Ctx      context.Context
	Profile  *profile.Profile
	Instance *Instance
	TmpDir   string
	PrevTs   func() time.Time
}

func newTestRunner(t *testing.T) *testRunner {
	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Time{} }

	tmpDir, err := ioutil.TempDir("", "lib_test_runner")
	if err != nil {
		t.Fatal(err)
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	return &testRunner{
		Ctx: context.Background(),
		// TODO (b5) - move test profile creation into testRunner constructor
		Profile:  testPeerProfile,
		Instance: NewInstanceFromConfigAndNode(config.DefaultConfigForTesting(), node),
		TmpDir:   tmpDir,
		PrevTs:   prevTs,
	}
}

func (tr *testRunner) Delete() {
	dsfs.Timestamp = tr.PrevTs
	os.RemoveAll(tr.TmpDir)
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

func (tr *testRunner) SaveDatasetFromBody(t *testing.T, dsName, bodyFilename string) dsref.Ref {
	m := NewDatasetMethods(tr.Instance)
	p := SaveParams{
		Ref:      fmt.Sprintf("peer/%s", dsName),
		BodyPath: bodyFilename,
	}
	r := reporef.DatasetRef{}
	if err := m.Save(&p, &r); err != nil {
		t.Fatal(err)
	}
	return reporef.ConvertToDsref(r)
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
