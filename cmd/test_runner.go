package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qri-io/ioes"
	ipfs_filestore "github.com/qri-io/qfs/cafs/ipfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	libtest "github.com/qri-io/qri/lib/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/gen"
	"github.com/spf13/cobra"
)

// TestRunner
type TestRunner struct {
	RepoRoot    *TestRepoRoot
	Context     context.Context
	ContextDone func()
	TsFunc      func() time.Time
	CmdR        *cobra.Command
	Teardown    func()
}

// NewTestRunner
func NewTestRunner(t *testing.T, peerName, testName string) *TestRunner {
	root := NewTestRepoRoot(t, peerName, testName)

	run := TestRunner{}
	run.RepoRoot = &root
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
	run.CmdR = run.RepoRoot.CreateCommandRunner(run.Context)
	return executeCommand(run.CmdR, cmdText)
}

// MustExec runs a command, returning standard output, failing the test if there's an error
func (run *TestRunner) MustExec(t *testing.T, cmdText string) string {
	if err := run.ExecCommand(cmdText); err != nil {
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

// GetCommandOutput returns the standard output from the previously executed command
func (run *TestRunner) GetCommandOutput() string {
	return run.RepoRoot.GetOutput()
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

// TODO: Perhaps this utility should move to a lower package, and be used as a way to validate the
// bodies of dataset in more of our test case. That would require extracting some parts out, like
// pathFactory, which would probably necessitate the pathFactory taking the testRepoRoot as a
// parameter to its constructor.

// TODO: Also, perhaps a different name would be better. This one is very similar to TestRepo,
// but does things quite differently.

// TestRepoRoot stores paths to a test repo.
type TestRepoRoot struct {
	rootPath    string
	ipfsPath    string
	qriPath     string
	pathFactory PathFactory
	testCrypto  gen.CryptoGenerator
	streams     ioes.IOStreams
	t           *testing.T
}

// NewTestRepoRoot constructs the test repo and initializes everything as cheaply as possible.
func NewTestRepoRoot(t *testing.T, peername, prefix string) TestRepoRoot {
	rootPath, err := ioutil.TempDir("", prefix)
	if err != nil {
		t.Fatal(err)
	}

	// Create directory for new IPFS repo.
	ipfsPath := filepath.Join(rootPath, "ipfs")
	err = os.MkdirAll(ipfsPath, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	// Build IPFS repo directory by unzipping an empty repo.
	testCrypto := libtest.NewTestCrypto()
	err = testCrypto.GenerateEmptyIpfsRepo(ipfsPath, "")
	if err != nil {
		t.Fatal(err)
	}
	// Create directory for new Qri repo.
	qriPath := filepath.Join(rootPath, "qri")
	err = os.MkdirAll(qriPath, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	// Create empty config.yaml into the test repo.
	cfg := config.DefaultConfigForTesting().Copy()
	cfg.Profile.Peername = peername
	err = cfg.WriteToFile(filepath.Join(qriPath, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// PathFactory returns the paths for qri and ipfs roots.
	pathFactory := NewDirPathFactory(rootPath)
	return TestRepoRoot{
		rootPath:    rootPath,
		ipfsPath:    ipfsPath,
		qriPath:     qriPath,
		pathFactory: pathFactory,
		testCrypto:  testCrypto,
		t:           t,
	}
}

// Delete removes the test repo on disk.
func (r *TestRepoRoot) Delete() {
	os.RemoveAll(r.rootPath)
}

// CreateCommandRunner returns a cobra runable command.
func (r *TestRepoRoot) CreateCommandRunner(ctx context.Context) *cobra.Command {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	r.streams = ioes.NewIOStreams(in, out, out)
	setNoColor(true)

	cmd := NewQriCommand(ctx, r.pathFactory, r.testCrypto, r.streams)
	cmd.SetOutput(out)
	return cmd
}

// GetOutput returns the output from the previously executed command.
func (r *TestRepoRoot) GetOutput() string {
	buffer, ok := r.streams.Out.(*bytes.Buffer)
	if ok {
		return buffer.String()
	}
	return ""
}

// GetPathForDataset returns the path to where the index'th dataset is stored on CAFS.
func (r *TestRepoRoot) GetPathForDataset(index int) string {
	dsRefs := filepath.Join(r.qriPath, "refs.fbs")

	data, err := ioutil.ReadFile(dsRefs)
	if err != nil {
		r.t.Fatal(err)
	}

	refs, err := repo.UnmarshalRefsFlatbuffer(data)
	if err != nil {
		r.t.Fatal(err)
	}

	// If dataset doesn't exist, return an empty string for the path.
	if len(refs) == 0 {
		return ""
	}

	return refs[index].Path
}

// ReadBodyFromIPFS reads the body of the dataset at the given keyPath stored in CAFS.
// TODO (b5): reprecate this rediculous function
func (r *TestRepoRoot) ReadBodyFromIPFS(keyPath string) string {
	ctx := context.Background()
	// TODO: Perhaps there is an existing cafs primitive that does this work instead?
	fs, err := ipfs_filestore.NewFilestore(func(cfg *ipfs_filestore.StoreCfg) {
		cfg.Online = false
		cfg.FsRepoPath = r.ipfsPath
	})
	if err != nil {
		r.t.Fatal(err)
	}

	bodyFile, err := fs.Get(ctx, keyPath)
	if err != nil {
		r.t.Fatal(err)
	}

	bodyBytes, err := ioutil.ReadAll(bodyFile)
	if err != nil {
		r.t.Fatal(err)
	}

	return string(bodyBytes)
}

// DatasetMarshalJSON reads the dataset head and marshals it as json.
func (r *TestRepoRoot) DatasetMarshalJSON(ref string) string {
	ctx := context.Background()
	fs, err := ipfs_filestore.NewFilestore(func(cfg *ipfs_filestore.StoreCfg) {
		cfg.Online = false
		cfg.FsRepoPath = r.ipfsPath
	})
	ds, err := dsfs.LoadDataset(ctx, fs, ref)
	if err != nil {
		r.t.Fatal(err)
	}
	bytes, err := json.Marshal(ds)
	if err != nil {
		r.t.Fatal(err)
	}
	return string(bytes)
}
