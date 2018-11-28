package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	libtest "github.com/qri-io/qri/lib/test"
	regmock "github.com/qri-io/registry/regserver/mock"
	"github.com/qri-io/qri/repo/gen"
	"github.com/spf13/cobra"
)

// ioReset resets the in, out, errs buffers
// convenience function used in testing
func ioReset(in, out, errs *bytes.Buffer) {
	in.Reset()
	out.Reset()
	errs.Reset()
}

func confirmQriNotRunning() error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", config.DefaultAPIPort))
	if err != nil {
		return fmt.Errorf("it looks like a qri server is already running on port %d, please close before running tests", config.DefaultAPIPort)
	}

	l.Close()
	return nil
}

func executeCommand(root *cobra.Command, cmd string) (output string, err error) {
	cmd = strings.TrimPrefix(cmd, "qri ")
	// WARNING - currently doesn't support quoted strings as input
	args := strings.Split(cmd, " ")
	_, output, err = executeCommandC(root, args...)
	return output, err
}

func executeCommandC(root *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	buf := &bytes.Buffer{}
	root.SetOutput(buf)
	root.SetArgs(args)

	c, err = root.ExecuteC()
	return c, buf.String(), err
}

const moviesCSVData = `movie_title,duration
Avatar,178
Pirates of the Caribbean: At World's End,169
Spectre,148
The Dark Knight Rises ,164
Star Wars: Episode VII - The Force Awakens,15
John Carter,132
Spider-Man 3,156
Tangled,100
Avengers: Age of Ultron,141`

const moviesCSVData2 = `movie_title,duration
Avatar,178
Pirates of the Caribbean: At World's End,169
Spectre,148
The Dark Knight Rises ,164
Star Wars: Episode VII - The Force Awakens,15
John Carter,132
Spider-Man 3,156
Tangled,100
Avengers: Age of Ultron,141
A Wild Film Appears!,2000
Another Film!,121`

const linksJSONData = `[
  "http://datatogether.org",
  "https://datatogether.org/css/style.css",
  "https://datatogether.org/img/favicon.ico",
  "https://datatogether.org",
  "https://datatogether.org/public-record",
  "https://datatogether.org/activities",
  "https://datatogether.org/activities/harvesting",
  "https://datatogether.org/activities/monitoring",
  "https://datatogether.org/activities/storing",
  "https://datatogether.org/activities/rescuing",
  "http://2017.code4lib.org",
  "https://datatogether.org/presentations/Code4Lib%202017%20-%20Golden%20Age%20for%20Libraries%20-%20Storing%20Data%20Together.pdf",
  "https://datatogether.org/presentations/Code4Lib%202017%20-%20Golden%20Age%20for%20Libraries%20-%20Storing%20Data%20Together.key",
  "http://www.esipfed.org/meetings/upcoming-meetings/esip-summer-meeting-2017",
  "https://datatogether.org/presentations/Data%20Together%20-%20ESIP%20Summer%20Meeting%20July%202017.pdf",
  "https://datatogether.org/presentations/Data%20Together%20-%20ESIP%20Summer%20Meeting%20July%202017.key",
  "https://archive.org/details/ndsr-dc-2017",
  "https://datatogether.org/presentations/Data%20Together%20-%20NDSR%20-%20swadeshi.pdf",
  "https://datatogether.org/presentations/Data%20Together%20-%20NDSR%20-%20swadeshi.key",
  "https://github.com/datatogether"
]`

const profileData = `
{
	"description" : "I'm a description!"
}
`

// This is a basic integration test that makes sure basic happy paths work on the CLI
func TestCommandsIntegration(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	_, registryServer := regmock.NewMockServer()

	path := filepath.Join(os.TempDir(), "qri_test_commands_integration")
	fmt.Printf("test filepath: %s\n", path)

	//clean up if previous cleanup failed
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.RemoveAll(path); err != nil {
			t.Fatalf("failed to cleanup from previous test execution: %s", err.Error())
		}
	}
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("error creating test path: %s", err.Error())
		return
	}
	defer os.RemoveAll(path)

	moviesFilePath := filepath.Join(path, "/movies.csv")
	if err := ioutil.WriteFile(moviesFilePath, []byte(moviesCSVData), os.ModePerm); err != nil {
		t.Errorf("error writing csv file: %s", err.Error())
		return
	}

	movies2FilePath := filepath.Join(path, "/movies2.csv")
	if err := ioutil.WriteFile(movies2FilePath, []byte(moviesCSVData2), os.ModePerm); err != nil {
		t.Errorf("error writing csv file: %s", err.Error())
		return
	}

	linksFilepath := filepath.Join(path, "/links.json")
	if err := ioutil.WriteFile(linksFilepath, []byte(linksJSONData), os.ModePerm); err != nil {
		t.Errorf("error writing json file: %s", err.Error())
		return
	}

	profileDataFilepath := filepath.Join(path, "/profile")
	if err := ioutil.WriteFile(profileDataFilepath, []byte(profileData), os.ModePerm); err != nil {
		t.Errorf("error profile json file: %s", err.Error())
		return
	}

	commands := []string{
		"qri help",
		"qri version",
		fmt.Sprintf("qri setup --peername=alan --registry=%s", registryServer.URL),
		"qri config get -c",
		"qri config get profile",
		"qri config set webapp.port 3505",
		fmt.Sprintf("qri save --body=%s me/movies", moviesFilePath),
		fmt.Sprintf("qri save --body=%s me/movies2", movies2FilePath),
		fmt.Sprintf("qri save --body=%s me/links", linksFilepath),
		"qri info me/movies",
		"qri list",
		fmt.Sprintf("qri save --body=%s -t=commit_1 me/movies", movies2FilePath),
		"qri log me/movies",
		"qri diff me/movies me/movies2 -d=detail",
		fmt.Sprintf("qri export -o=%s me/movies", path),
		fmt.Sprintf("qri export -o=%s --format=cbor --body-format=json me/movies", path),
		"qri publish me/movies",
		"qri ls -p",
		"qri publish --unpublish me/movies",
		// TODO - currently removed, see TODO in cmd/registry.go
		// "qri registry unpublish me/movies",
		// "qri registry publish me/movies",
		"qri rename me/movies me/movie",
		"qri body --limit=1 --format=cbor me/movie",
		"qri validate me/movie",
		"qri remove me/movie",
		fmt.Sprintf("qri export --blank -o=%s/blank_dataset.yaml", path),
		"qri setup --remove",
	}

	streams, _, _, _ := ioes.NewTestIOStreams()
	root := NewQriCommand(NewDirPathFactory(path), libtest.NewTestCrypto(), streams)

	for i, command := range commands {
		func() {
			defer func() {
				if e := recover(); e != nil {
					fmt.Println(e)
					t.Errorf("case %d unexpected panic executing command\n%s\n%s", i, command, e)
					return
				}
			}()

			_, er := executeCommand(root, command)
			if er != nil {
				fmt.Println(er)
				t.Errorf("case %d unexpected error executing command\n%s\n%s", i, command, er.Error())
				return
			}

			time.Sleep(500 * time.Millisecond)
		}()
	}
}

// Test that saving a dataset with a relative body path works, and validate the contents of that
// body match what was given to the save command.
func TestSaveRelativeBodyPath(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	r := NewTestRepoRoot(t, "qri_test_save_relative_body")
	defer r.Delete()

	// TODO: If TestRepoRoot is moved to a different package, pass it an a parameter to this
	// function.
	cmdR := r.CreateCommandRunner()
	_, err := executeCommand(cmdR, "qri save --file=testdata/movies/ds_ten.yaml me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Read body from the dataset that was saved.
	dsPath := r.GetPathForDataset(0)
	actualBody := r.ReadBodyFromIPFS(dsPath + "/data.csv")

	// Read the body from the testdata input file.
	f, _ := os.Open("testdata/movies/body_ten.csv")
	expectBytes, _ := ioutil.ReadAll(f)
	expectBody := string(expectBytes)

	// Make sure they match.
	if actualBody != expectBody {
		t.Errorf("error reading body, expect \"%s\", actual \"%s\"", actualBody, expectBody)
	}
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
	t           *testing.T
}

// NewTestRepoRoot constructs the test repo and initializes everything as cheaply as possible.
func NewTestRepoRoot(t *testing.T, prefix string) TestRepoRoot {
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
	cfg := config.DefaultConfigForTesting()
	cfg.Profile.Peername = "test_peer"
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
func (r *TestRepoRoot) CreateCommandRunner() *cobra.Command {
	streams, _, _, _ := ioes.NewTestIOStreams()
	return NewQriCommand(r.pathFactory, r.testCrypto, streams)
}

// GetPathForDataset returns the path to where the index'th dataset is stored on CAFS.
func (r *TestRepoRoot) GetPathForDataset(index int) string {
	dsRefs := filepath.Join(r.qriPath, "ds_refs.json")
	file, err := os.Open(dsRefs)
	if err != nil {
		r.t.Fatal(err)
	}

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		r.t.Fatal(err)
	}

	var result []map[string]interface{}
	err = json.Unmarshal([]byte(bytes), &result)
	if err != nil {
		r.t.Fatal(err)
	}

	var dsPath string
	dsPath = result[index]["path"].(string)
	return dsPath
}

// ReadBodyFromIPFS reads the body of the dataset at the given keyPath stored in CAFS.
func (r *TestRepoRoot) ReadBodyFromIPFS(keyPath string) string {
	// TODO: Perhaps there is an existing cafs primitive that does this work instead?
	fs, err := ipfs_filestore.NewFilestore(func(cfg *ipfs_filestore.StoreCfg) {
		cfg.Online = false
		cfg.FsRepoPath = r.ipfsPath
	})
	if err != nil {
		r.t.Fatal(err)
	}

	bodyFile, err := fs.Get(datastore.NewKey(keyPath))
	if err != nil {
		r.t.Fatal(err)
	}

	bodyBytes, err := ioutil.ReadAll(bodyFile)
	if err != nil {
		r.t.Fatal(err)
	}

	return string(bodyBytes)
}
