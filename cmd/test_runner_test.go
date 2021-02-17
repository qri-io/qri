package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"text/scanner"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/gen"
	reporef "github.com/qri-io/qri/repo/ref"
	repotest "github.com/qri-io/qri/repo/test"
	tfrun "github.com/qri-io/qri/transform/run"
	"github.com/qri-io/qri/transform/startf"
	"github.com/spf13/cobra"
)

// TestRunner holds data used to run tests
type TestRunner struct {
	RepoRoot *repotest.TempRepo
	RepoPath string

	Context       context.Context
	ContextDone   func()
	TmpDir        string
	Streams       ioes.IOStreams
	InStream      *bytes.Buffer
	OutStream     *bytes.Buffer
	ErrStream     *bytes.Buffer
	DsfsTsFunc    func() time.Time
	LogbookTsFunc func() int64
	LocOrig       *time.Location
	XformVersion  string
	CmdR          *cobra.Command
	Teardown      func()
	CmdDoneCh     chan struct{}
	TestCrypto    gen.CryptoGenerator

	Registry *registry.Registry
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

// NewTestRunnerUsingPeerInfoWithMockRemoteClient constructs a test runner using an
// explicit testPeer, as well as a mock remote client
func NewTestRunnerUsingPeerInfoWithMockRemoteClient(t *testing.T, peerInfoNum int, peerName, testName string) *TestRunner {
	root, err := repotest.NewTempRepoUsingPeerInfo(peerInfoNum, peerName, testName)
	if err != nil {
		t.Fatalf("creating temp repo: %s", err)
	}
	root.UseMockRemoteClient = true
	return newTestRunnerFromRoot(&root)
}

// NewTestRunnerWithTempRegistry constructs a test runner with a mock registry connection
func NewTestRunnerWithTempRegistry(t *testing.T, peerName, testName string) *TestRunner {
	t.Helper()
	root, err := repotest.NewTempRepoFixedProfileID(peerName, testName)
	if err != nil {
		t.Fatalf("creating temp repo: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	// TODO(dustmop): Switch to root.TestCrypto. Until then, we're reusing the
	// same testPeers, leading to different nodes with the same profileID
	g := repotest.NewTestCrypto()
	reg, teardownRegistry, err := regserver.NewTempRegistry(ctx, "registry", testName+"_registry", g)
	if err != nil {
		t.Fatalf("creating registry: %s", err)
	}

	// TODO (b5) - wouldn't it be nice if we could pass the client as an instance configuration
	// option? that'd require re-thinking the way we do NewQriCommand
	_, server := regserver.NewMockServerRegistry(*reg)

	runner := newTestRunnerFromRoot(&root)
	runner.Registry = reg
	prevTeardown := runner.Teardown
	runner.Teardown = func() {
		cancel()
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

func useConsistentRunIDs() {
	source := strings.NewReader(strings.Repeat("OmgZombies!?!?!", 200))
	tfrun.SetIDRand(source)
}

func newTestRunnerFromRoot(root *repotest.TempRepo) *TestRunner {
	ctx, cancel := context.WithCancel(context.Background())
	useConsistentRunIDs()

	run := TestRunner{
		RepoRoot:    root,
		RepoPath:    filepath.Join(root.RootPath, "qri"),
		Context:     ctx,
		ContextDone: cancel,
		TestCrypto:  root.TestCrypto,
	}

	// TmpDir will be removed recursively, only if it is non-empty
	run.TmpDir = ""

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	dsfsCounter := 0
	run.DsfsTsFunc = dsfs.Timestamp
	dsfs.Timestamp = func() time.Time {
		dsfsCounter++
		return time.Date(2001, 01, 01, 01, dsfsCounter, 01, 01, time.UTC)
	}

	// Do the same for logbook.NewTimestamp
	bookCounter := 0
	run.LogbookTsFunc = logbook.NewTimestamp
	logbook.NewTimestamp = func() int64 {
		bookCounter++
		return time.Date(2001, 01, 01, 01, bookCounter, 01, 01, time.UTC).Unix()
	}

	// Set IOStreams
	run.Streams, run.InStream, run.OutStream, run.ErrStream = ioes.NewTestIOStreams()
	setNoColor(true)
	//setNoPrompt(true)

	// Set the location to New York so that timezone printing is consistent
	location, _ := time.LoadLocation("America/New_York")
	run.LocOrig = StringerLocation
	StringerLocation = location

	// Stub the version of starlark, because it is output when transforms run
	run.XformVersion = startf.Version
	startf.Version = "test_version"

	return &run
}

// Delete cleans up after a TestRunner is done being used.
func (run *TestRunner) Delete() {
	if run.Teardown != nil {
		run.Teardown()
	}
	if run.TmpDir != "" {
		os.RemoveAll(run.TmpDir)
	}
	// restore random RunID generator
	tfrun.SetIDRand(nil)
	dsfs.Timestamp = run.DsfsTsFunc
	logbook.NewTimestamp = run.LogbookTsFunc
	StringerLocation = run.LocOrig
	startf.Version = run.XformVersion
	run.ContextDone()
	run.RepoRoot.Delete()
}

// MakeTmpDir returns a temporary directory that runner will delete when done
func (run *TestRunner) MakeTmpDir(t *testing.T, pattern string) string {
	if run.TmpDir != "" {
		t.Fatal("can only make one tmpDir at a time")
	}
	tmpDir, err := ioutil.TempDir("", pattern)
	if err != nil {
		t.Fatal(err)
	}
	run.TmpDir = tmpDir
	return tmpDir
}

// TODO(dustmop): Look into using options instead of multiple exec functions.

// ExecCommand executes the given command string
func (run *TestRunner) ExecCommand(cmdText string) error {
	var shutdown func() <-chan error
	run.CmdR, shutdown = run.CreateCommandRunner(run.Context)
	if err := executeCommand(run.CmdR, cmdText); err != nil {
		timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
		return err
	}

	return timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
}

// ExecCommandWithStdin executes the given command string with the string as stdin content
func (run *TestRunner) ExecCommandWithStdin(ctx context.Context, cmdText, stdinText string) error {
	setNoColor(true)
	run.Streams.In = strings.NewReader(stdinText)
	cmd, shutdown := NewQriCommand(ctx, run.RepoPath, run.RepoRoot.TestCrypto, run.Streams)
	cmd.SetOutput(run.OutStream)
	run.CmdR = cmd
	if err := executeCommand(run.CmdR, cmdText); err != nil {
		return err
	}

	return timedShutdown(fmt.Sprintf("ExecCommandCombinedOutErr: %q\n", cmdText), shutdown)
}

// ExecCommandCombinedOutErr executes the command with a combined stdout and stderr stream
func (run *TestRunner) ExecCommandCombinedOutErr(cmdText string) error {
	ctx, cancel := context.WithCancel(run.Context)
	var shutdown func() <-chan error
	run.CmdR, shutdown = run.CreateCommandRunnerCombinedOutErr(ctx)
	if err := executeCommand(run.CmdR, cmdText); err != nil {
		cancel()
		return err
	}

	err := timedShutdown(fmt.Sprintf("ExecCommandCombinedOutErr: %q\n", cmdText), shutdown)
	cancel()
	return err
}

func timedShutdown(cmd string, shutdown func() <-chan error) error {
	waitForDone := make(chan error)
	go func() {
		select {
		case <-time.NewTimer(time.Second * 3).C:
			waitForDone <- fmt.Errorf("%s shutdown didn't send on 'done' channel within 3 seconds of context cancellation", cmd)
		case err := <-shutdown():
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				err = nil
			}
			waitForDone <- err
		}
	}()
	return <-waitForDone
}

func shutdownRepoGraceful(cancel context.CancelFunc, r repo.Repo) error {
	var (
		wg  sync.WaitGroup
		err error
	)
	wg.Add(1)

	go func() {
		<-r.Done()
		err = r.DoneErr()
		wg.Done()
	}()
	cancel()
	wg.Wait()
	return err
}

// ExecCommandWithContext executes the given command with a context
func (run *TestRunner) ExecCommandWithContext(ctx context.Context, cmdText string) error {
	var shutdown func() <-chan error
	run.CmdR, shutdown = run.CreateCommandRunner(ctx)
	if err := executeCommand(run.CmdR, cmdText); err != nil {
		return err
	}

	return timedShutdown(fmt.Sprintf("ExecCommandWithContext: %q\n", cmdText), shutdown)
}

func (run *TestRunner) MustExecuteQuotedCommand(t *testing.T, quotedCmdText string) string {
	var shutdown func() <-chan error
	run.CmdR, shutdown = run.CreateCommandRunner(run.Context)

	if err := executeQuotedCommand(run.CmdR, quotedCmdText); err != nil {
		_, callerFile, callerLine, ok := runtime.Caller(1)
		if !ok {
			t.Fatal(err)
		} else {
			t.Fatalf("%s:%d: %s", callerFile, callerLine, err)
		}
	}
	if err := timedShutdown(fmt.Sprintf("MustExecuteQuotedCommand: %q\n", quotedCmdText), shutdown); err != nil {
		t.Error(err)
	}
	return run.GetCommandOutput()
}

// CreateCommandRunner returns a cobra runable command.
func (run *TestRunner) CreateCommandRunner(ctx context.Context) (*cobra.Command, func() <-chan error) {
	return run.newCommandRunner(ctx, false)
}

// CreateCommandRunnerCombinedOutErr returns a cobra command that combines stdout and stderr
func (run *TestRunner) CreateCommandRunnerCombinedOutErr(ctx context.Context) (*cobra.Command, func() <-chan error) {
	cmd, shutdown := run.newCommandRunner(ctx, true)
	return cmd, shutdown
}

func (run *TestRunner) newCommandRunner(ctx context.Context, combineOutErr bool) (*cobra.Command, func() <-chan error) {
	run.IOReset()
	streams := run.Streams
	if combineOutErr {
		streams = ioes.NewIOStreams(run.InStream, run.OutStream, run.OutStream)
	}
	if run.RepoRoot.UseMockRemoteClient {
		// Set this context value, which is used in lib.NewInstance to construct a
		// remote.MockClient instead. Using context.Value is discouraged, but it's difficult
		// to pipe parameters into cobra.Command without doing it like this.
		key := lib.InstanceContextKey("RemoteClient")
		ctx = context.WithValue(ctx, key, "mock")
	}
	cmd, shutdown := NewQriCommand(ctx, run.RepoPath, run.RepoRoot.TestCrypto, streams)
	cmd.SetOutput(run.OutStream)
	return cmd, shutdown
}

// Username returns the test username from the config's profile
func (run *TestRunner) Username() string {
	return run.RepoRoot.GetConfig().Profile.Peername
}

// IOReset resets the io streams
func (run *TestRunner) IOReset() {
	run.InStream.Reset()
	run.OutStream.Reset()
	run.ErrStream.Reset()
}

// FileExists returns whether the file exists
func (run *TestRunner) FileExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

// LookupVersionInfo returns a versionInfo for the ref, or nil if not found
func (run *TestRunner) LookupVersionInfo(t *testing.T, refStr string) *dsref.VersionInfo {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := run.RepoRoot.Repo(ctx)
	if err != nil {
		t.Fatal(err)
	}

	dr, err := dsref.Parse(refStr)
	if err != nil {
		t.Fatal(err)
	}

	// TODO(b5): me shortcut is handled by an instance, it'd be nice we had a
	// function in the repo package that deduplicated this in both places
	if dr.Username == "me" {
		pro, err := r.Profile(ctx)
		if err != nil {
			t.Fatal(err)
		}
		dr.Username = pro.Peername
	}

	if _, err := r.ResolveRef(ctx, &dr); err != nil {
		return nil
	}

	// TODO(b5): TestUnlinkNoHistory relies on a nil-return versionInfo, so
	// we need to ignore this error for now
	vi, _ := repo.GetVersionInfoShim(r, dr)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// TODO(b5) - hand-creating a shutdown function to satisfy "timedshutdown",
	// which works with an instance in most other cases
	shutdown := func() <-chan error {
		finished := make(chan error)
		go func() {
			<-r.Done()
			finished <- r.DoneErr()
		}()

		cancel()
		return finished
	}

	err = timedShutdown("LookupVersionInfo", shutdown)
	if err != nil {
		t.Fatal(err)
	}

	return vi
}

// ClearFSIPath clears the FSIPath for a reference in the refstore
func (run *TestRunner) ClearFSIPath(t *testing.T, refStr string) {
	dr, err := dsref.Parse(refStr)
	if err != nil {
		t.Fatal(err)
	}
	datasetRef := reporef.RefFromDsref(dr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := run.RepoRoot.Repo(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.CanonicalizeDatasetRef(ctx, r, &datasetRef)
	if err != nil {
		t.Fatal(err)
	}
	datasetRef.FSIPath = ""
	r.PutRef(datasetRef)

	shutdown := func() <-chan error {
		finished := make(chan error)
		go func() {
			<-r.Done()
			finished <- r.DoneErr()
		}()

		return finished
	}

	cancel()
	if err := timedShutdown("ClearFSIPath", shutdown); err != nil {
		t.Fatal(err)
	}
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
func (run *TestRunner) ReadBodyFromIPFS(t *testing.T, path string) string {
	body, err := run.RepoRoot.ReadBodyFromIPFS(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// DatasetMarshalJSON reads the dataset head and marshals it as json
func (run *TestRunner) DatasetMarshalJSON(t *testing.T, ref string) string {
	data, err := run.RepoRoot.DatasetMarshalJSON(ref)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// MustLoadDataset loads the dataset or fails
func (run *TestRunner) MustLoadDataset(t *testing.T, ref string) *dataset.Dataset {
	ds, err := run.RepoRoot.LoadDataset(ref)
	if err != nil {
		t.Fatal(err)
	}
	return ds
}

// MustExec runs a command, returning standard output, failing the test if there's an error
func (run *TestRunner) MustExec(t *testing.T, cmdText string) string {
	if err := run.ExecCommand(cmdText); err != nil {
		_, callerFile, callerLine, ok := runtime.Caller(1)
		if !ok {
			t.Fatal(err)
		} else {
			t.Fatalf("executing command: %q\n%s:%d: %s", cmdText, callerFile, callerLine, err)
		}
	}
	return run.GetCommandOutput()
}

// MustExec runs a command, returning combined standard output and standard err
func (run *TestRunner) MustExecCombinedOutErr(t *testing.T, cmdText string) string {
	t.Helper()
	ctx, cancel := context.WithCancel(run.Context)
	var shutdown func() <-chan error
	run.CmdR, shutdown = run.CreateCommandRunnerCombinedOutErr(ctx)
	err := executeCommand(run.CmdR, cmdText)
	if err != nil {
		cancel()
		_, callerFile, callerLine, ok := runtime.Caller(1)
		if !ok {
			t.Fatal(err)
		} else {
			t.Fatalf("%s:%d: %s", callerFile, callerLine, err)
		}
	}

	err = timedShutdown("MustExecCombinedOutErr", shutdown)
	cancel()
	if err != nil {
		t.Fatal(err)
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
	outputText := ""
	if buffer, ok := run.Streams.Out.(*bytes.Buffer); ok {
		outputText = run.niceifyTempDirs(buffer.String())
	}
	return outputText
}

// GetCommandErrOutput fetches the stderr value from the previously executed
// command
func (run *TestRunner) GetCommandErrOutput() string {
	errOutText := ""
	if buffer, ok := run.Streams.ErrOut.(*bytes.Buffer); ok {
		errOutText = run.niceifyTempDirs(buffer.String())
	}
	return errOutText
}

func (run *TestRunner) niceifyTempDirs(text string) string {
	text = strings.Replace(text, run.RepoRoot.RootPath, "/root", -1)
	realRoot, err := filepath.EvalSymlinks(run.RepoRoot.RootPath)
	if err == nil {
		text = strings.Replace(text, realRoot, "/root", -1)
	}
	return text
}

func executeCommand(root *cobra.Command, cmd string) error {
	// fmt.Printf("exec command: %s\n", cmd)
	cmd = strings.TrimPrefix(cmd, "qri ")
	args := strings.Split(cmd, " ")
	return executeCommandC(root, args...)
}

func executeQuotedCommand(root *cobra.Command, cmd string) error {
	cmd = strings.TrimPrefix(cmd, "qri ")

	var s scanner.Scanner
	s.Init(strings.NewReader(cmd))
	var args []string
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		arg := s.TokenText()
		if unquoted, err := strconv.Unquote(arg); err == nil {
			arg = unquoted
		}

		args = append(args, arg)
	}

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
func (run *TestRunner) AddDatasetToRefstore(t *testing.T, refStr string, ds *dataset.Dataset) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ref, err := dsref.ParseHumanFriendly(refStr)
	if err != nil && err != dsref.ErrBadCaseName {
		t.Fatal(err)
	}

	ds.Peername = ref.Username
	ds.Name = ref.Name

	r, err := run.RepoRoot.Repo(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Reserve the name in the logbook, which provides an initID
	initID, err := r.Logbook().WriteDatasetInit(ctx, ds.Name)
	if err != nil {
		t.Fatal(err)
	}

	// No existing commit
	emptyHeadRef := ""

	if _, err = base.SaveDataset(ctx, r, r.Filesystem().DefaultWriteFS(), initID, emptyHeadRef, ds, nil, base.SaveSwitches{}); err != nil {
		t.Fatal(err)
	}

	cancel()
	<-r.Done()
}
