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
	"github.com/qri-io/qri/auth/key"
	run "github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver"
	remotemock "github.com/qri-io/qri/remote/mock"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
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
	TestCrypto    key.CryptoGenerator

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

	tr := newTestRunnerFromRoot(&root)
	tr.Registry = reg
	prevTeardown := tr.Teardown
	tr.Teardown = func() {
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

	return tr
}

func useConsistentRunIDs() {
	source := strings.NewReader(strings.Repeat("OmgZombies!?!?!", 200))
	run.SetIDRand(source)
}

func newTestRunnerFromRoot(root *repotest.TempRepo) *TestRunner {
	ctx, cancel := context.WithCancel(context.Background())
	useConsistentRunIDs()

	tr := TestRunner{
		RepoRoot:    root,
		RepoPath:    filepath.Join(root.RootPath, "qri"),
		Context:     ctx,
		ContextDone: cancel,
		TestCrypto:  root.TestCrypto,
	}

	// TmpDir will be removed recursively, only if it is non-empty
	tr.TmpDir = ""

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	dsfsCounter := 0
	tr.DsfsTsFunc = dsfs.Timestamp
	dsfs.Timestamp = func() time.Time {
		dsfsCounter++
		return time.Date(2001, 01, 01, 01, dsfsCounter, 01, 01, time.UTC)
	}

	// Do the same for logbook.NewTimestamp
	bookCounter := 0
	tr.LogbookTsFunc = logbook.NewTimestamp
	logbook.NewTimestamp = func() int64 {
		bookCounter++
		return time.Date(2001, 01, 01, 01, bookCounter, 01, 01, time.UTC).Unix()
	}

	// Set IOStreams
	tr.Streams, tr.InStream, tr.OutStream, tr.ErrStream = ioes.NewTestIOStreams()
	setNoColor(true)
	//setNoPrompt(true)

	// Set the location to New York so that timezone printing is consistent
	location, _ := time.LoadLocation("America/New_York")
	tr.LocOrig = StringerLocation
	StringerLocation = location

	// Stub the version of starlark, because it is output when transforms run
	tr.XformVersion = startf.Version
	startf.Version = "test_version"

	return &tr
}

// Delete cleans up after a TestRunner is done being used.
func (runner *TestRunner) Delete() {
	if runner.Teardown != nil {
		runner.Teardown()
	}
	if runner.TmpDir != "" {
		os.RemoveAll(runner.TmpDir)
	}
	// restore random RunID generator
	run.SetIDRand(nil)
	dsfs.Timestamp = runner.DsfsTsFunc
	logbook.NewTimestamp = runner.LogbookTsFunc
	StringerLocation = runner.LocOrig
	startf.Version = runner.XformVersion
	runner.ContextDone()
	runner.RepoRoot.Delete()
}

// MakeTmpDir returns a temporary directory that runner will delete when done
func (runner *TestRunner) MakeTmpDir(t *testing.T, pattern string) string {
	if runner.TmpDir != "" {
		t.Fatal("can only make one tmpDir at a time")
	}
	tmpDir, err := ioutil.TempDir("", pattern)
	if err != nil {
		t.Fatal(err)
	}
	runner.TmpDir = tmpDir
	return tmpDir
}

// TODO(dustmop): Look into using options instead of multiple exec functions.

// ExecCommand executes the given command string
func (runner *TestRunner) ExecCommand(cmdText string) error {
	var shutdown func() <-chan error
	runner.CmdR, shutdown = runner.CreateCommandRunner(runner.Context)
	if err := executeCommand(runner.CmdR, cmdText); err != nil {
		timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
		return err
	}

	return timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
}

// ExecCommandWithStdin executes the given command string with the string as stdin content
func (runner *TestRunner) ExecCommandWithStdin(ctx context.Context, cmdText, stdinText string) error {
	setNoColor(true)
	runner.Streams.In = strings.NewReader(stdinText)
	cmd, shutdown := NewQriCommand(ctx, runner.RepoPath, runner.RepoRoot.TestCrypto, runner.Streams)
	cmd.SetOutput(runner.OutStream)
	runner.CmdR = cmd
	if err := executeCommand(runner.CmdR, cmdText); err != nil {
		return err
	}

	return timedShutdown(fmt.Sprintf("ExecCommandWithStdin: %q\n", cmdText), shutdown)
}

// ExecCommandCombinedOutErr executes the command with a combined stdout and stderr stream
func (runner *TestRunner) ExecCommandCombinedOutErr(cmdText string) error {
	ctx, cancel := context.WithCancel(runner.Context)
	var shutdown func() <-chan error
	runner.CmdR, shutdown = runner.CreateCommandRunnerCombinedOutErr(ctx)
	if err := executeCommand(runner.CmdR, cmdText); err != nil {
		shutDownErr := <-shutdown()
		if shutDownErr != nil {
			log.Errorf("error shutting down %q: %q", cmdText, shutDownErr)
		}
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
func (runner *TestRunner) ExecCommandWithContext(ctx context.Context, cmdText string) error {
	var shutdown func() <-chan error
	runner.CmdR, shutdown = runner.CreateCommandRunner(ctx)
	if err := executeCommand(runner.CmdR, cmdText); err != nil {
		return err
	}

	return timedShutdown(fmt.Sprintf("ExecCommandWithContext: %q\n", cmdText), shutdown)
}

func (runner *TestRunner) MustExecuteQuotedCommand(t *testing.T, quotedCmdText string) string {
	var shutdown func() <-chan error
	runner.CmdR, shutdown = runner.CreateCommandRunner(runner.Context)

	if err := executeQuotedCommand(runner.CmdR, quotedCmdText); err != nil {
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
	return runner.GetCommandOutput()
}

// CreateCommandRunner returns a cobra runable command.
func (runner *TestRunner) CreateCommandRunner(ctx context.Context) (*cobra.Command, func() <-chan error) {
	return runner.newCommandRunner(ctx, false)
}

// CreateCommandRunnerCombinedOutErr returns a cobra command that combines stdout and stderr
func (runner *TestRunner) CreateCommandRunnerCombinedOutErr(ctx context.Context) (*cobra.Command, func() <-chan error) {
	cmd, shutdown := runner.newCommandRunner(ctx, true)
	return cmd, shutdown
}

func (runner *TestRunner) newCommandRunner(ctx context.Context, combineOutErr bool) (*cobra.Command, func() <-chan error) {
	runner.IOReset()
	streams := runner.Streams
	if combineOutErr {
		streams = ioes.NewIOStreams(runner.InStream, runner.OutStream, runner.OutStream)
	}
	var opts []lib.Option
	if runner.RepoRoot.UseMockRemoteClient {
		opts = append(opts, lib.OptRemoteClientConstructor(remotemock.NewClient))
	}
	cmd, shutdown := NewQriCommand(ctx, runner.RepoPath, runner.RepoRoot.TestCrypto, streams, opts...)
	cmd.SetOutput(runner.OutStream)
	return cmd, shutdown
}

// Username returns the test username from the config's profile
func (runner *TestRunner) Username() string {
	return runner.RepoRoot.GetConfig().Profile.Peername
}

// IOReset resets the io streams
func (runner *TestRunner) IOReset() {
	runner.InStream.Reset()
	runner.OutStream.Reset()
	runner.ErrStream.Reset()
}

// FileExists returns whether the file exists
func (runner *TestRunner) FileExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

// LookupVersionInfo returns a versionInfo for the ref, or nil if not found
func (runner *TestRunner) LookupVersionInfo(t *testing.T, refStr string) *dsref.VersionInfo {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := runner.RepoRoot.Repo(ctx)
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
		dr.Username = r.Profiles().Owner().Peername
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
// Note: ClearFSIPAth doesn't do reference resoultion, cannot use "me" in
// dataset names
func (runner *TestRunner) ClearFSIPath(t *testing.T, refStr string) {
	t.Helper()
	dr, err := dsref.Parse(refStr)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := runner.RepoRoot.Repo(ctx)
	if err != nil {
		t.Fatal(err)
	}

	vi, err := repo.GetVersionInfoShim(r, dr)
	if err != nil {
		t.Fatal(err)
	}
	vi.FSIPath = ""
	if err := repo.PutVersionInfoShim(ctx, r, vi); err != nil {
		t.Fatal(err)
	}

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
func (runner *TestRunner) GetPathForDataset(t *testing.T, index int) string {
	path, err := runner.RepoRoot.GetPathForDataset(index)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// ReadBodyFromIPFS reads body data from an IPFS repo by path string,
func (runner *TestRunner) ReadBodyFromIPFS(t *testing.T, path string) string {
	body, err := runner.RepoRoot.ReadBodyFromIPFS(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// DatasetMarshalJSON reads the dataset head and marshals it as json
func (runner *TestRunner) DatasetMarshalJSON(t *testing.T, ref string) string {
	data, err := runner.RepoRoot.DatasetMarshalJSON(ref)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// MustLoadDataset loads the dataset or fails
func (runner *TestRunner) MustLoadDataset(t *testing.T, ref string) *dataset.Dataset {
	ds, err := runner.RepoRoot.LoadDataset(ref)
	if err != nil {
		t.Fatal(err)
	}
	return ds
}

// MustExec runs a command, returning standard output, failing the test if there's an error
func (runner *TestRunner) MustExec(t *testing.T, cmdText string) string {
	if err := runner.ExecCommand(cmdText); err != nil {
		_, callerFile, callerLine, ok := runtime.Caller(1)
		if !ok {
			t.Fatal(err)
		} else {
			t.Fatalf("executing command: %q\n%s:%d: %s", cmdText, callerFile, callerLine, err)
		}
	}
	return runner.GetCommandOutput()
}

// MustExec runs a command, returning combined standard output and standard err
func (runner *TestRunner) MustExecCombinedOutErr(t *testing.T, cmdText string) string {
	t.Helper()
	ctx, cancel := context.WithCancel(runner.Context)
	var shutdown func() <-chan error
	runner.CmdR, shutdown = runner.CreateCommandRunnerCombinedOutErr(ctx)
	err := executeCommand(runner.CmdR, cmdText)
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
	return runner.GetCommandOutput()
}

// MustWriteFile writes to a file, failing the test if there's an error
func (runner *TestRunner) MustWriteFile(t *testing.T, filename, contents string) {
	if err := ioutil.WriteFile(filename, []byte(contents), os.FileMode(0644)); err != nil {
		t.Fatal(err)
	}
}

// MustReadFile reads a file, failing the test if there's an error
func (runner *TestRunner) MustReadFile(t *testing.T, filename string) string {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes)
}

// Must asserts that the function result passed to it is not an error
func (runner *TestRunner) Must(t *testing.T, err error) {
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
func (runner *TestRunner) GetCommandOutput() string {
	outputText := ""
	if buffer, ok := runner.Streams.Out.(*bytes.Buffer); ok {
		outputText = runner.niceifyTempDirs(buffer.String())
	}
	return outputText
}

// GetCommandErrOutput fetches the stderr value from the previously executed
// command
func (runner *TestRunner) GetCommandErrOutput() string {
	errOutText := ""
	if buffer, ok := runner.Streams.ErrOut.(*bytes.Buffer); ok {
		errOutText = runner.niceifyTempDirs(buffer.String())
	}
	return errOutText
}

func (runner *TestRunner) niceifyTempDirs(text string) string {
	text = strings.Replace(text, runner.RepoRoot.RootPath, "/root", -1)
	realRoot, err := filepath.EvalSymlinks(runner.RepoRoot.RootPath)
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
func (runner *TestRunner) AddDatasetToRefstore(t *testing.T, refStr string, ds *dataset.Dataset) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ref, err := dsref.ParseHumanFriendly(refStr)
	if err != nil && err != dsref.ErrBadCaseName {
		t.Fatal(err)
	}

	ds.Peername = ref.Username
	ds.Name = ref.Name

	r, err := runner.RepoRoot.Repo(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// WARNING: here we're assuming the provided ref matches the owner peername
	author := r.Logbook().Owner()

	// Reserve the name in the logbook, which provides an initID
	initID, err := r.Logbook().WriteDatasetInit(ctx, author, ds.Name)
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
