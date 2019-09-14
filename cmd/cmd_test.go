package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/ioes"
	ipfs_filestore "github.com/qri-io/qfs/cafs/ipfs"
	"github.com/qri-io/qri/config"
	libtest "github.com/qri-io/qri/lib/test"
	regmock "github.com/qri-io/qri/registry/regserver/mock"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/gen"
	"github.com/spf13/cobra"
)

func init() {
	// TODO (b5) - ask go-ipfs folks if the shutdown messages can be INFO level
	// instead of error level to avoid:
	// 10:12:42.396 ERROR       core: core is shutting down...
	// after all sorts of tests
	golog.SetLogLevel("core", "CRITICAL")
}

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

func confirmUpdateServiceNotRunning() error {
	l, err := net.Listen("tcp", config.DefaultUpdateAddress)
	if err != nil {
		return fmt.Errorf("it looks like a qri update service is already running on port %d, please close before running tests", config.DefaultAPIPort)
	}

	l.Close()
	return nil
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
	// fmt.Printf("test filepath: %s\n", path)

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
		"qri list",
		"qri list me/movies",
		fmt.Sprintf("qri save --body=%s -t=commit_1 me/movies", movies2FilePath),
		"qri log me/movies",
		"qri diff me/movies me/movies2",
		fmt.Sprintf("qri export -o=%s me/movies --zip", path),
		fmt.Sprintf("qri export -o=%s/ds.yaml --format=yaml me/movies", path),
		"qri publish me/movies",
		"qri ls -p",
		"qri publish --unpublish me/movies",
		// TODO - currently removed, see TODO in cmd/registry.go
		// "qri registry unpublish me/movies",
		// "qri registry publish me/movies",
		"qri rename me/movies me/movie",
		"qri get body --page-size=1 --format=cbor me/movie",
		"qri validate me/movie",
		"qri remove me/movie --revisions=all",
		fmt.Sprintf("qri export --blank -o=%s/blank_dataset.yaml", path),
		"qri setup --remove",
	}

	streams, _, _, _ := ioes.NewTestIOStreams()
	ctx, done := context.WithCancel(context.Background())
	defer done()

	root := NewQriCommand(ctx, NewDirPathFactory(path), libtest.NewTestCrypto(), streams)
	root.SetOutput(ioutil.Discard)

	for i, command := range commands {
		func() {
			defer func() {
				if e := recover(); e != nil {
					t.Errorf("case %d unexpected panic executing command\n%s\n%s", i, command, e)
					return
				}
			}()

			er := executeCommand(root, command)
			if er != nil {
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

	ctx, done := context.WithCancel(context.Background())
	defer done()

	// TODO: If TestRepoRoot is moved to a different package, pass it an a parameter to this
	// function.
	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --file=testdata/movies/ds_ten.yaml me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Read body from the dataset that was saved.
	dsPath := r.GetPathForDataset(0)
	actualBody := r.ReadBodyFromIPFS(dsPath + "/body.csv")

	// Read the body from the testdata input file.
	f, _ := os.Open("testdata/movies/body_ten.csv")
	expectBytes, _ := ioutil.ReadAll(f)
	expectBody := string(expectBytes)

	// Make sure they match.
	if actualBody != expectBody {
		t.Errorf("error reading body, expect \"%s\", actual \"%s\"", actualBody, expectBody)
	}
}

// Test that saving three revisions, then removing the newest two, leaves the first body.
func TestRemoveOnlyTwoRevisions(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	r := NewTestRepoRoot(t, "qri_test_remove_only_one_revision")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri remove me/test_movies --revisions=2")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Read body from the dataset that was saved.
	dsPath := r.GetPathForDataset(0)
	actualBody := r.ReadBodyFromIPFS(dsPath + "/body.csv")

	// Read the body from the testdata input file.
	f, _ := os.Open("testdata/movies/body_ten.csv")
	expectBytes, _ := ioutil.ReadAll(f)
	expectBody := string(expectBytes)

	// Make sure they match.
	if expectBody != actualBody {
		t.Errorf("error reading body, expect \"%s\", actual \"%s\"", expectBody, actualBody)
	}
}

// Test that adding three revision, then removing all of them leaves nothing.
func TestRemoveAllRevisionsLongForm(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	r := NewTestRepoRoot(t, "qri_test_remove_only_one_revision")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri remove me/test_movies --revisions=all")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Read path for dataset, which shouldn't exist anymore.
	dsPath := r.GetPathForDataset(0)
	if dsPath != "" {
		t.Errorf("expected dataset to be removed entirely, found at \"%s\"", dsPath)
	}
}

// Test that adding three revision, then removing all of them leaves nothing, using --all.
func TestRemoveAllRevisionsShortForm(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	r := NewTestRepoRoot(t, "qri_test_remove_only_one_revision")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri remove me/test_movies --all")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Read path for dataset, which shouldn't exist anymore.
	dsPath := r.GetPathForDataset(0)
	if dsPath != "" {
		t.Errorf("expected dataset to be removed entirely, found at \"%s\"", dsPath)
	}
}

// Test that save can override a single component, meta in this case.
func TestSaveThenOverrideMetaComponent(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r := NewTestRepoRoot(t, "qri_test_save_then_override_meta")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --file=testdata/movies/meta_override.yaml me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Read head from the dataset that was saved, as json string.
	dsPath := r.GetPathForDataset(0)
	actual := r.DatasetMarshalJSON(dsPath)

	// This dataset is ds_ten.yaml, with the meta replaced by meta_override.yaml.
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"\t- modified title\n","path":"/ipfs/QmfEiBVM5MLdw2zBgFVMcCRjavFmna8cLnPQiRzkjmWPBu","qri":"cm:0","signature":"I/nrDkgwt1IPtdFKvgMQAIRYvOqKfqm6x0qfpuJ14rEtO3+uPnY3K5pVDMWJ7K+pYJz6fyguYWgXHKkbo5wZl0ICVyoIiPa9zIVbqc1d6j1v13WqtRb0bn1CXQvuI6HcBhb7+VqkSW1m+ALpxhNQuI4ZfRv8Nm8MbEpL6Ct55fJpWX1zszJ2rQP1LcH2AlEZ8bl0qpcFMk03LENUHSt1DjlaApxrEJzDgAs5drfndxXgGKYjPpkjdF+qGhn2ALV2tC64I5aIn1SJPAQnVwprUr1FmVZjZcF9m9r8WnzQ6ldj29eZIciiFlT4n2Cbw+dgPo/hNRsgzn7Our2a6r5INw==","timestamp":"2001-01-01T01:01:01.000000001Z","title":"Meta: 1 change"},"meta":{"qri":"md:0","title":"different title"},"path":"/ipfs/QmSpXgpakANF3c4Z7qEZDv5tqmTw1r7Jyefg7NCJtJrohv","peername":"me","previousPath":"/ipfs/QmdxjWGrjc9neXqReY6bHMC4eGG5je358PcCWCHNVYbLGU","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmXkN5J5yCAtF8GCxwRXARzAQhj3bPaSv1VHoyCCXzQRzN","scriptPath":"/ipfs/QmVM37PFzBcZn3qqKvyQ9rJ1jC8NkS8kYZNJke1Wje1jor"}}`
	if actual != expect {
		t.Errorf("error, dataset actual:\n%s\nexpect:\n%s\n", actual, expect)
	}
}

// Test that saving with two components at once will merge them together.
func TestSaveTwoComponents(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r := NewTestRepoRoot(t, "qri_test_save_then_override_meta")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --file=testdata/movies/meta_override.yaml --file=testdata/movies/structure_override.json me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Read head from the dataset that was saved, as json string.
	dsPath := r.GetPathForDataset(0)
	actual := r.DatasetMarshalJSON(dsPath)

	// This dataset is ds_ten.yaml, with the meta replaced by meta_override ("different title") and
	// the structure replaced by structure_override (lazyQuotes: false && title: "name").
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"\t- modified formatConfig\n\t- modified schema\n","path":"/ipfs/QmbrZGpqFJZjW8JPZMZUQ8zDRZU3QakGE6hQ9D4tgvZtYx","qri":"cm:0","signature":"I/nrDkgwt1IPtdFKvgMQAIRYvOqKfqm6x0qfpuJ14rEtO3+uPnY3K5pVDMWJ7K+pYJz6fyguYWgXHKkbo5wZl0ICVyoIiPa9zIVbqc1d6j1v13WqtRb0bn1CXQvuI6HcBhb7+VqkSW1m+ALpxhNQuI4ZfRv8Nm8MbEpL6Ct55fJpWX1zszJ2rQP1LcH2AlEZ8bl0qpcFMk03LENUHSt1DjlaApxrEJzDgAs5drfndxXgGKYjPpkjdF+qGhn2ALV2tC64I5aIn1SJPAQnVwprUr1FmVZjZcF9m9r8WnzQ6ldj29eZIciiFlT4n2Cbw+dgPo/hNRsgzn7Our2a6r5INw==","timestamp":"2001-01-01T01:01:01.000000001Z","title":"Structure: 2 changes"},"meta":{"qri":"md:0","title":"different title"},"path":"/ipfs/QmVTLs5CkSfz9cczUqLXFXjypxEYZW6UKin7RXuGgKbusT","peername":"me","previousPath":"/ipfs/QmdxjWGrjc9neXqReY6bHMC4eGG5je358PcCWCHNVYbLGU","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":false},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"name","type":"string"},{"title":"duration","type":"integer"}]},"type":"array"}},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmXkN5J5yCAtF8GCxwRXARzAQhj3bPaSv1VHoyCCXzQRzN","scriptPath":"/ipfs/QmVM37PFzBcZn3qqKvyQ9rJ1jC8NkS8kYZNJke1Wje1jor"}}`
	if actual != expect {
		t.Errorf("error, dataset actual:\n%s\nexpect:\n%s\n", actual, expect)
	}
}

// Test that save can override just the transform
func TestSaveThenOverrideTransform(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r := NewTestRepoRoot(t, "qri_test_save_file_transform")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --file=testdata/movies/tf.star me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Read head from the dataset that was saved, as json string.
	dsPath := r.GetPathForDataset(0)
	actual := r.DatasetMarshalJSON(dsPath)

	// This dataset is ds_ten.yaml, with an added transform section
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"\t- modified scriptPath\n\t- modified syntax\n\t- ...\n...modified syntaxVersion","path":"/ipfs/QmRhDyCTYDEshqscC7LHrmcGMBE7bVX9osY7MfB9RDyJ1J","qri":"cm:0","signature":"I/nrDkgwt1IPtdFKvgMQAIRYvOqKfqm6x0qfpuJ14rEtO3+uPnY3K5pVDMWJ7K+pYJz6fyguYWgXHKkbo5wZl0ICVyoIiPa9zIVbqc1d6j1v13WqtRb0bn1CXQvuI6HcBhb7+VqkSW1m+ALpxhNQuI4ZfRv8Nm8MbEpL6Ct55fJpWX1zszJ2rQP1LcH2AlEZ8bl0qpcFMk03LENUHSt1DjlaApxrEJzDgAs5drfndxXgGKYjPpkjdF+qGhn2ALV2tC64I5aIn1SJPAQnVwprUr1FmVZjZcF9m9r8WnzQ6ldj29eZIciiFlT4n2Cbw+dgPo/hNRsgzn7Our2a6r5INw==","timestamp":"2001-01-01T01:01:01.000000001Z","title":"Transform: 3 changes"},"meta":{"qri":"md:0","title":"example movie data"},"path":"/ipfs/Qme8JCANgfN77eH8Lq82bLBVQrLibWxxRdqsLpV4vxiKhW","peername":"me","previousPath":"/ipfs/QmdxjWGrjc9neXqReY6bHMC4eGG5je358PcCWCHNVYbLGU","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"transform":{"qri":"tf:0","scriptPath":"/ipfs/Qmb69tx5VCL7q7EfkGKpDgESBysmDbohoLvonpbgri48NN","syntax":"starlark","syntaxVersion":"0.8.1"},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmXkN5J5yCAtF8GCxwRXARzAQhj3bPaSv1VHoyCCXzQRzN","scriptPath":"/ipfs/QmVM37PFzBcZn3qqKvyQ9rJ1jC8NkS8kYZNJke1Wje1jor"}}`
	if actual != expect {
		t.Errorf("error, dataset actual:\n%s\nexpect:\n%s\n", actual, expect)
	}
}

// Test that save can override just the viz
func TestSaveThenOverrideViz(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r := NewTestRepoRoot(t, "qri_test_save_file_transform")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --file=testdata/template.html me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Read head from the dataset that was saved, as json string.
	dsPath := r.GetPathForDataset(0)
	actual := r.DatasetMarshalJSON(dsPath)

	// This dataset is ds_ten.yaml, with an added viz section
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"\t- modified scriptPath\n","path":"/ipfs/QmZfiXv9QcMWHEF7S6HBwQhqGGEDhiHRVhXwh6oE9si44Q","qri":"cm:0","signature":"I/nrDkgwt1IPtdFKvgMQAIRYvOqKfqm6x0qfpuJ14rEtO3+uPnY3K5pVDMWJ7K+pYJz6fyguYWgXHKkbo5wZl0ICVyoIiPa9zIVbqc1d6j1v13WqtRb0bn1CXQvuI6HcBhb7+VqkSW1m+ALpxhNQuI4ZfRv8Nm8MbEpL6Ct55fJpWX1zszJ2rQP1LcH2AlEZ8bl0qpcFMk03LENUHSt1DjlaApxrEJzDgAs5drfndxXgGKYjPpkjdF+qGhn2ALV2tC64I5aIn1SJPAQnVwprUr1FmVZjZcF9m9r8WnzQ6ldj29eZIciiFlT4n2Cbw+dgPo/hNRsgzn7Our2a6r5INw==","timestamp":"2001-01-01T01:01:01.000000001Z","title":"Viz: 1 change"},"meta":{"qri":"md:0","title":"example movie data"},"path":"/ipfs/QmNrhFKYf1KfqGT11hx6enH8whNJ8dVwDomt78wgGW8pXU","peername":"me","previousPath":"/ipfs/QmdxjWGrjc9neXqReY6bHMC4eGG5je358PcCWCHNVYbLGU","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmVrEH7T7XmdJLym8YL9DjwCALbz264h7GQTrjkSGmbvry","scriptPath":"/ipfs/QmRaVGip3V9fVBJheZN6FbUajD3ZLNjHhXdjrmfg2JPoo5"}}`
	if actual != expect {
		t.Errorf("error, dataset actual:\n%s\nexpect:\n%s\n", actual, expect)
	}
}

// Test that save can combine a meta compoent, and a transform, and a viz
func TestSaveThenOverrideMetaAndTransformAndViz(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r := NewTestRepoRoot(t, "qri_test_save_file_transform")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --file=testdata/movies/meta_override.yaml --file=testdata/movies/tf.star --file=testdata/template.html me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Read head from the dataset that was saved, as json string.
	dsPath := r.GetPathForDataset(0)
	actual := r.DatasetMarshalJSON(dsPath)

	// This dataset is ds_ten.yaml, with an added meta component, and transform, and viz
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"\t- modified scriptPath\n\t- modified syntax\n\t- ...\n...modified syntaxVersion","path":"/ipfs/QmRhDyCTYDEshqscC7LHrmcGMBE7bVX9osY7MfB9RDyJ1J","qri":"cm:0","signature":"I/nrDkgwt1IPtdFKvgMQAIRYvOqKfqm6x0qfpuJ14rEtO3+uPnY3K5pVDMWJ7K+pYJz6fyguYWgXHKkbo5wZl0ICVyoIiPa9zIVbqc1d6j1v13WqtRb0bn1CXQvuI6HcBhb7+VqkSW1m+ALpxhNQuI4ZfRv8Nm8MbEpL6Ct55fJpWX1zszJ2rQP1LcH2AlEZ8bl0qpcFMk03LENUHSt1DjlaApxrEJzDgAs5drfndxXgGKYjPpkjdF+qGhn2ALV2tC64I5aIn1SJPAQnVwprUr1FmVZjZcF9m9r8WnzQ6ldj29eZIciiFlT4n2Cbw+dgPo/hNRsgzn7Our2a6r5INw==","timestamp":"2001-01-01T01:01:01.000000001Z","title":"Transform: 3 changes"},"meta":{"qri":"md:0","title":"different title"},"path":"/ipfs/QmPjMymsos45XKeMaBg1D4ajYtdFQxP1epqmAsEdLBjKyd","peername":"me","previousPath":"/ipfs/QmdxjWGrjc9neXqReY6bHMC4eGG5je358PcCWCHNVYbLGU","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"transform":{"qri":"tf:0","scriptPath":"/ipfs/Qmb69tx5VCL7q7EfkGKpDgESBysmDbohoLvonpbgri48NN","syntax":"starlark","syntaxVersion":"0.8.1"},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmVrEH7T7XmdJLym8YL9DjwCALbz264h7GQTrjkSGmbvry","scriptPath":"/ipfs/QmRaVGip3V9fVBJheZN6FbUajD3ZLNjHhXdjrmfg2JPoo5"}}`
	if actual != expect {
		t.Errorf("error, dataset actual:\n%s\nexpect:\n%s\n", actual, expect)
	}
}

// Test that saving a full dataset with a component at the same time is an error
func TestSaveDatasetWithComponentError(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r := NewTestRepoRoot(t, "qri_test_save_then_override_meta")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --file=testdata/movies/ds_ten.yaml --file=testdata/movies/meta_override.yaml me/test_ds")
	if err == nil {
		t.Errorf("expected error, did not get one")
	}

	expect := `conflict, cannot save a full dataset with other components`
	if err.Error() != expect {
		t.Errorf("expected error: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

// Test that saving with two components of the same kind is an error
func TestSaveConflictingComponents(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r := NewTestRepoRoot(t, "qri_test_save_then_override_meta")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --file=testdata/movies/meta_override.yaml --file=testdata/movies/meta_override.yaml me/test_ds")
	if err == nil {
		t.Errorf("expected error, did not get one")
	}

	expect := `conflict, multiple components of kind "md"`
	if err.Error() != expect {
		t.Errorf("expected error: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

// Test that running a transform without any changes will not make a new commit
func TestSaveTransformWithoutChanges(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r := NewTestRepoRoot(t, "qri_test_transform_same")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --file=testdata/movies/tf_123.star me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --file=testdata/movies/tf_123.star me/test_ds")
	expect := `error saving: no changes detected`
	if err == nil {
		t.Errorf("expected error: did not get one")
	}
	if err.Error() != expect {
		t.Errorf("expected error: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

// Test that calling `get_body` will retrieve the body of the previous version.
func TestTransformUsingGetBodyAndSetBody(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r := NewTestRepoRoot(t, "qri_test_save_transform_get_body")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --body=testdata/movies/body_two.json me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --file=testdata/movies/tf_add_one.star me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Read body from the dataset that was created with the transform
	dsPath := r.GetPathForDataset(0)
	actualBody := r.ReadBodyFromIPFS(dsPath + "/body.json")

	// This body is body_two.json, with the numbers in the second column increased by 1.
	expectBody := `[["Avatar",179],["Pirates of the Caribbean: At World's End",170]]`
	if actualBody != expectBody {
		t.Errorf("error, dataset actual:\n%s\nexpect:\n%s\n", actualBody, expectBody)
	}
}

// Test that we can compare bodies of different dataset revisions.
func TestDiffRevisions(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r := NewTestRepoRoot(t, "qri_test_diff_revisions")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri diff body me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	output := r.GetOutput()
	expect := `+30 elements. 30 inserts. 0 deletes. 0 updates.

+ 18: ["Dragonfly ",104]
+ 19: ["The Black Dahlia ",121]
+ 20: ["Flyboys ",140]
+ 21: ["The Last Castle ",131]
+ 22: ["Supernova ",91]
+ 23: ["Winter's Tale ",118]
+ 24: ["The Mortal Instruments: City of Bones ",130]
+ 25: ["Meet Dave ",90]
+ 26: ["Dark Water ",103]
+ 27: ["Edtv ",122]
`
	if output != expect {
		t.Errorf("error, did not match actual:\n\"%v\"\nexpect:\n\"%v\"\n", output, expect)
	}
}

// Test that diffing a dataset with only one version produces an error
func TestDiffOnlyOneRevision(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r := NewTestRepoRoot(t, "qri_test_diff_only_one")
	defer r.Delete()

	ctx, done := context.WithCancel(context.Background())
	defer done()

	cmdR := r.CreateCommandRunner(ctx)
	err := executeCommand(cmdR, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cmdR = r.CreateCommandRunner(ctx)
	err = executeCommand(cmdR, "qri diff body me/test_movies")
	if err == nil {
		t.Errorf("expected error, did not get one")
	}

	expect := `dataset has only one version, nothing to diff against`
	if err.Error() != expect {
		t.Errorf("expected error: \"%s\", got: \"%s\"", expect, err.Error())
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
	streams     ioes.IOStreams
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
	cfg := config.DefaultConfigForTesting().Copy()
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
