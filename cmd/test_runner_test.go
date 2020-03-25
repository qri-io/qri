package cmd

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
	repotest "github.com/qri-io/qri/repo/test"
	"github.com/spf13/cobra"
)

// TestRunner holds data used to run tests
type TestRunner struct {
	RepoRoot    *repotest.TempRepo
	Context     context.Context
	ContextDone func()
	TsFunc      func() time.Time
	CmdR        *cobra.Command
	Teardown    func()

	pathFactory PathFactory

	registry *registry.Registry
}

// NewTestRunner constructs a new TestRunner
func NewTestRunner(t *testing.T, peerName, testName string) *TestRunner {
	root, err := repotest.NewTempRepoFixedProfileID(peerName, testName)
	if err != nil {
		t.Fatalf("creating temp repo: %s", err)
	}
	return newTestRunnerFromRoot(&root)
}

// NewTestRunnerWithMockRemoteClient constructs a test runner with a mock remote client
func NewTestRunnerWithMockRemoteClient(t *testing.T, peerName, testName string) *TestRunner {
	root, err := repotest.NewTempRepoFixedProfileID(peerName, testName)
	if err != nil {
		t.Fatalf("creating temp repo: %s", err)
	}
	root.UseMockRemoteClient = true
	return newTestRunnerFromRoot(&root)
}

// NewTestRunnerWithTempRegistry constructs a test runner with a mock registry connection
func NewTestRunnerWithTempRegistry(t *testing.T, peerName, testName string) *TestRunner {
	root, err := repotest.NewTempRepoFixedProfileID(peerName, testName)
	if err != nil {
		t.Fatalf("creating temp repo: %s", err)
	}

	g := repotest.NewTestCrypto()
	reg, teardownRegistry, err := regserver.NewTempRegistry("registry", testName+"_registry", g)
	if err != nil {
		t.Fatalf("creating registry: %s", err)
	}

	// TODO (b5) - wouldn't it be nice if we could pass the client as an instance configuration
	// option? that'd require re-thinking the way we do NewQriCommand
	_, server := regserver.NewMockServerRegistry(*reg)

	runner := newTestRunnerFromRoot(&root)
	prevTeardown := runner.Teardown
	runner.Teardown = func() {
		teardownRegistry()
		server.Close()
		if prevTeardown != nil {
			prevTeardown()
		}
	}

	root.GetConfig().Registry.Location = server.URL
	if err := root.WriteConfigFile(); err != nil {
		t.Fatalf("writing config file: %s", err)
	}

	return runner
}

func newTestRunnerFromRoot(root *repotest.TempRepo) *TestRunner {
	run := TestRunner{
		pathFactory: NewDirPathFactory(root.RootPath),
	}
	run.RepoRoot = root
	run.Context, run.ContextDone = context.WithCancel(context.Background())

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	counter := 0
	run.TsFunc = dsfs.Timestamp
	dsfs.Timestamp = func() time.Time {
		counter++
		return time.Date(2001, 01, 01, 01, counter, 01, 01, time.UTC)
	}

	return &run
}

// Delete cleans up after a TestRunner is done being used.
func (run *TestRunner) Delete() {
	if run.Teardown != nil {
		run.Teardown()
	}
	dsfs.Timestamp = run.TsFunc
	run.ContextDone()
	run.RepoRoot.Delete()
}

// ExecCommand executes the given command string
func (run *TestRunner) ExecCommand(cmdText string) error {
	run.CmdR = run.CreateCommandRunner(run.Context)
	return executeCommand(run.CmdR, cmdText)
}

// ExecCommandWithStdin executes the given command string with the string as stdin content
func (run *TestRunner) ExecCommandWithStdin(ctx context.Context, cmdText, stdinText string) error {
	in := bytes.NewBufferString(stdinText)
	out := &bytes.Buffer{}
	run.RepoRoot.Streams = ioes.NewIOStreams(in, out, out)
	setNoColor(true)
	cmd := NewQriCommand(ctx, run.pathFactory, run.RepoRoot.TestCrypto, run.RepoRoot.Streams)
	cmd.SetOutput(out)
	run.CmdR = cmd
	return executeCommand(run.CmdR, cmdText)
}

// ExecCommandWithContext executes the given command with a context
func (run *TestRunner) ExecCommandWithContext(ctx context.Context, cmdText string) error {
	run.CmdR = run.CreateCommandRunner(ctx)
	return executeCommand(run.CmdR, cmdText)
}

// CreateCommandRunner returns a cobra runable command.
func (run *TestRunner) CreateCommandRunner(ctx context.Context) *cobra.Command {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	run.RepoRoot.Streams = ioes.NewIOStreams(in, out, out)
	setNoColor(true)

	if run.RepoRoot.UseMockRemoteClient {
		// Set this context value, which is used in lib.NewInstance to construct a
		// remote.MockClient instead. Using context.Value is discouraged, but it's difficult
		// to pipe parameters into cobra.Command without doing it like this.
		key := lib.InstanceContextKey("RemoteClient")
		ctx = context.WithValue(ctx, key, "mock")
	}

	cmd := NewQriCommand(ctx, run.pathFactory, run.RepoRoot.TestCrypto, run.RepoRoot.Streams)
	cmd.SetOutput(out)
	return cmd
}

// FileExists returns whether the file exists
func (run *TestRunner) FileExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

// LookupVersionInfo returns a versionInfo for the ref, or nil if not found
func (run *TestRunner) LookupVersionInfo(refStr string) *dsref.VersionInfo {
	// TODO(dustmop): Could directly parse reporef.DatasetRef instead, but we should transition
	// to dsref's data structures where possible. This will make it easier to switch to dscache
	// once it exists.
	dr, err := dsref.Parse(refStr)
	if err != nil {
		return nil
	}
	datasetRef := reporef.RefFromDsref(dr)
	// TODO(dustmop): Work-around for https://github.com/qri-io/qri/issues/1209
	// Would rather do `run.RepoRoot.Repo()` but that doesn't work.
	ctx := context.Background()
	inst, err := lib.NewInstance(
		ctx,
		run.RepoRoot.QriPath,
		lib.OptStdIOStreams(),
		lib.OptSetIPFSPath(run.RepoRoot.IPFSPath),
	)
	r := inst.Repo()
	err = repo.CanonicalizeDatasetRef(r, &datasetRef)
	if err != nil {
		return nil
	}
	vinfo := reporef.ConvertToVersionInfo(&datasetRef)
	return &vinfo
}

// ClearFSIPath clears the FSIPath for a reference in the refstore
func (run *TestRunner) ClearFSIPath(t *testing.T, refStr string) {
	dr, err := dsref.Parse(refStr)
	if err != nil {
		t.Fatal(err)
	}
	datasetRef := reporef.RefFromDsref(dr)
	// TODO(dustmop): Work-around for https://github.com/qri-io/qri/issues/1209
	// Would rather do `run.RepoRoot.Repo()` but that doesn't work.
	ctx := context.Background()
	inst, err := lib.NewInstance(
		ctx,
		run.RepoRoot.QriPath,
		lib.OptStdIOStreams(),
		lib.OptSetIPFSPath(run.RepoRoot.IPFSPath),
	)
	r := inst.Repo()
	err = repo.CanonicalizeDatasetRef(r, &datasetRef)
	if err != nil {
		t.Fatal(err)
	}
	datasetRef.FSIPath = ""
	r.PutRef(datasetRef)
}

// GetPathForDataset fetches a path for dataset index
func (run *TestRunner) GetPathForDataset(t *testing.T, index int) string {
	path, err := run.RepoRoot.GetPathForDataset(index)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// ReadBodyFromIPFS reads body data from an IPFS repo by path string,
func (run *TestRunner) ReadBodyFromIPFS(t *testing.T, path string) (body string) {
	body, err := run.RepoRoot.ReadBodyFromIPFS(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// DatasetMarshalJSON reads the dataset head and marshals it as json
func (run *TestRunner) DatasetMarshalJSON(t *testing.T, ref string) (data string) {
	data, err := run.RepoRoot.DatasetMarshalJSON(ref)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// MustExec runs a command, returning standard output, failing the test if there's an error
func (run *TestRunner) MustExec(t *testing.T, cmdText string) string {
	if err := run.ExecCommand(cmdText); err != nil {
		_, callerFile, callerLine, ok := runtime.Caller(1)
		if !ok {
			t.Fatal(err)
		} else {
			t.Fatalf("%s:%d: %s", callerFile, callerLine, err)
		}
	}
	return run.GetCommandOutput()
}

// MustWriteFile writes to a file, failing the test if there's an error
func (run *TestRunner) MustWriteFile(t *testing.T, filename, contents string) {
	if err := ioutil.WriteFile(filename, []byte(contents), os.FileMode(0644)); err != nil {
		t.Fatal(err)
	}
}

// MustReadFile reads a file, failing the test if there's an error
func (run *TestRunner) MustReadFile(t *testing.T, filename string) string {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes)
}

// Must asserts that the function result passed to it is not an error
func (run *TestRunner) Must(t *testing.T, err error) {
	if err != nil {
		_, callerFile, callerLine, ok := runtime.Caller(1)
		if !ok {
			t.Fatal(err)
		} else {
			t.Fatalf("%s:%d: %s", callerFile, callerLine, err)
		}
	}
}

// GetCommandOutput returns the standard output from the previously executed
// command
func (run *TestRunner) GetCommandOutput() string {
	return run.RepoRoot.GetOutput()
}

// GetCommandErrOutput fetches the stderr value from the previously executed
// command
func (run *TestRunner) GetCommandErrOutput() string {
	return run.RepoRoot.GetErrOutput()
}

func executeCommand(root *cobra.Command, cmd string) error {
	cmd = strings.TrimPrefix(cmd, "qri ")
	// WARNING - currently doesn't support quoted strings as input
	args := strings.Split(cmd, " ")
	return executeCommandC(root, args...)
}

func executeCommandC(root *cobra.Command, args ...string) (err error) {
	root.SetArgs(args)
	_, err = root.ExecuteC()
	return err
}

// AddDatasetToRefstore adds a dataset to the test runner's refstore. It ignores the upper-levels
// of our stack, namely cmd/ and lib/, which means it can be used to add a dataset with a name
// that is using upper-case characters.
func (run *TestRunner) AddDatasetToRefstore(ctx context.Context, t *testing.T, refStr string, ds *dataset.Dataset) {
	ref, err := dsref.ParseHumanFriendly(refStr)
	if err != nil && err != dsref.ErrBadCaseName {
		t.Fatal(err)
	}

	ds.Peername = ref.Username
	ds.Name = ref.Name

	inst, err := lib.NewInstance(ctx, run.RepoRoot.QriPath)
	// NOTE(dustmop): There's a bug with TestRepo that I don't understand completely. The commands
	// are run using a different refstore than the refstore returned by accessing the fields of the
	// TestRepo directly. The command runner constructs a repo and then refstore which has a path
	// similar to "/var/folders/tmpDir/T/qri_save_bad_case1234" with "qri" and "ipfs" directories
	// within. However, trying to directly access the Repo object from TestRepo will return a
	// refstore with the path "/var/folders/tmpDir/T/qri_save_bad_case1234" as the *qri repository*.
	//
	// So doing:
	//   run.RepoRoot.Repo()
	// gives a refstore that saves to:
	//   "/var/folders/tmpDir/T/qri_save_bad_case1234/refs.fbs"
	// While the commandRunner is using:
	//   "/var/folders/tmpDir/T/qri_save_bad_case1234/qri/refs.fbs"
	//
	// We work around this by constructing a lib.Instance, which uses the PathFactory to get the
	// qri subfolder and correctly use the refstore at:
	//   "/var/folders/tmpDir/T/qri_save_bad_case1234/qri/refs.fbs"
	//
	// This is probably the same bug that is handled in repo/buildrepo/build.go with a hack that
	// appends "/qri" to the repoPath.
	r := inst.Repo()

	str := ioes.NewStdIOStreams()
	secrets := make(map[string]string)
	scriptOut := &bytes.Buffer{}
	sw := base.SaveDatasetSwitches{}

	_, err = base.SaveDataset(ctx, r, str, ds, secrets, scriptOut, sw)
	if err != nil {
		t.Fatal(err)
	}
}
