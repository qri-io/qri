package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	libtest "github.com/qri-io/qri/lib/test"
	regmock "github.com/qri-io/qri/registry/regserver/mock"
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

	run := NewTestRunner(t, "test_peer", "qri_test_save_relative_body")
	defer run.Delete()

	// Save a dataset which has a body as a relative path
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_movies")

	// Read body from the dataset that was saved.
	dsPath := run.RepoRoot.GetPathForDataset(0)
	actualBody := run.RepoRoot.ReadBodyFromIPFS(dsPath + "/body.csv")

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

	run := NewTestRunner(t, "test_peer", "qri_test_remove_only_two_revisions")
	defer run.Delete()

	// Save three revisions, then remove two
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	run.MustExec(t, "qri remove me/test_movies --revisions=2")

	// Read body from the dataset that was saved.
	dsPath := run.RepoRoot.GetPathForDataset(0)
	actualBody := run.RepoRoot.ReadBodyFromIPFS(dsPath + "/body.csv")

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

	run := NewTestRunner(t, "test_peer", "qri_test_remove_only_one_revision")
	defer run.Delete()

	// Save three versions, then remove all of them.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	run.MustExec(t, "qri remove me/test_movies --revisions=all")

	// Read path for dataset, which shouldn't exist anymore.
	dsPath := run.RepoRoot.GetPathForDataset(0)
	if dsPath != "" {
		t.Errorf("expected dataset to be removed entirely, found at \"%s\"", dsPath)
	}
}

// Test that adding three revision, then removing all of them leaves nothing, using --all.
func TestRemoveAllRevisionsShortForm(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_remove_only_one_revision")
	defer run.Delete()

	// Save three versions, then remove all of them, using the --all flag.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	run.MustExec(t, "qri remove me/test_movies --all")

	// Read path for dataset, which shouldn't exist anymore.
	dsPath := run.RepoRoot.GetPathForDataset(0)
	if dsPath != "" {
		t.Errorf("expected dataset to be removed entirely, found at \"%s\"", dsPath)
	}
}

// Test that save can override a single component, meta in this case.
func TestSaveThenOverrideMetaComponent(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_then_override_meta")
	defer run.Delete()

	// Save a version, then save another with a new meta component.
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/movies/meta_override.yaml me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.RepoRoot.GetPathForDataset(0)
	actual := run.RepoRoot.DatasetMarshalJSON(dsPath)

	// This dataset is ds_ten.yaml, with the meta replaced by meta_override.yaml.
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"\t- modified title\n","path":"/ipfs/QmZQFezXryWYXqEFkSfpqNMY2w9LMyAxNMxARwNT36FutV","qri":"cm:0","signature":"njCFxpGqq0xJSrjgxC289KncjflqA0e00txweEqIyUTvEKSUBKHcfQmx4OQIJzJqQJdcjIEzFrwP9cdquozRgsnrpsSfKb+wBWdtbnrg8zfat0X/Dqjro6JD7afJf0gU9s5SDi/s8g/qZOLwWh1nuoH4UAeUX+l3DH0ocFjeD6r/YkMJ0KXaWaFloKP8UPasfqoei9PxxmYQuAnFMqpXFisB7mKFAbgbpF3eL80UcbQPTih7WF11SBym/AzJhGNvOivOjmRxKGEuqEH9g3NPTEQr+LnP415X4qiaZA6MVmOO66vC0diUN4vJUMvhTsWnVEBtgqjTRYlSaYwabHv/gA==","timestamp":"2001-01-01T01:02:01.000000001Z","title":"Meta: 1 change"},"meta":{"qri":"md:0","title":"different title"},"path":"/ipfs/QmaYjcYAEMNUEQTgCNGaPg3yaSEQ3SmHPegAxVZTfWWWJM","peername":"me","previousPath":"/ipfs/QmdxjWGrjc9neXqReY6bHMC4eGG5je358PcCWCHNVYbLGU","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmXkN5J5yCAtF8GCxwRXARzAQhj3bPaSv1VHoyCCXzQRzN","scriptPath":"/ipfs/QmVM37PFzBcZn3qqKvyQ9rJ1jC8NkS8kYZNJke1Wje1jor"}}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("dataset (-want +got):\n%s", diff)
	}
}

// Test that saving with two components at once will merge them together.
func TestSaveTwoComponents(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_then_override_meta")
	defer run.Delete()

	// Save a version, then same another with two components at once
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/movies/meta_override.yaml --file=testdata/movies/structure_override.json me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.RepoRoot.GetPathForDataset(0)
	actual := run.RepoRoot.DatasetMarshalJSON(dsPath)

	// This dataset is ds_ten.yaml, with the meta replaced by meta_override ("different title") and
	// the structure replaced by structure_override (lazyQuotes: false && title: "name").
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"\t- modified formatConfig\n\t- modified schema\n","path":"/ipfs/QmcrLQftiH7RGyyKCTr5YzSso5pK6FBf2oZusdDNNNeSHJ","qri":"cm:0","signature":"njCFxpGqq0xJSrjgxC289KncjflqA0e00txweEqIyUTvEKSUBKHcfQmx4OQIJzJqQJdcjIEzFrwP9cdquozRgsnrpsSfKb+wBWdtbnrg8zfat0X/Dqjro6JD7afJf0gU9s5SDi/s8g/qZOLwWh1nuoH4UAeUX+l3DH0ocFjeD6r/YkMJ0KXaWaFloKP8UPasfqoei9PxxmYQuAnFMqpXFisB7mKFAbgbpF3eL80UcbQPTih7WF11SBym/AzJhGNvOivOjmRxKGEuqEH9g3NPTEQr+LnP415X4qiaZA6MVmOO66vC0diUN4vJUMvhTsWnVEBtgqjTRYlSaYwabHv/gA==","timestamp":"2001-01-01T01:02:01.000000001Z","title":"Structure: 2 changes"},"meta":{"qri":"md:0","title":"different title"},"path":"/ipfs/QmWNkcBdohvFaQG6GquGbZwq2LnuyDaojAMAfSmFDH3qRr","peername":"me","previousPath":"/ipfs/QmdxjWGrjc9neXqReY6bHMC4eGG5je358PcCWCHNVYbLGU","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":false},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"name","type":"string"},{"title":"duration","type":"integer"}]},"type":"array"}},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmXkN5J5yCAtF8GCxwRXARzAQhj3bPaSv1VHoyCCXzQRzN","scriptPath":"/ipfs/QmVM37PFzBcZn3qqKvyQ9rJ1jC8NkS8kYZNJke1Wje1jor"}}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("dataset (-want +got):\n%s", diff)
	}
}

// Test that save can override just the transform
func TestSaveThenOverrideTransform(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_file_transform")
	defer run.Delete()

	// Save a version, then save another with a transform
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/movies/tf.star me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.RepoRoot.GetPathForDataset(0)
	actual := run.RepoRoot.DatasetMarshalJSON(dsPath)

	// This dataset is ds_ten.yaml, with an added transform section
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"\t- modified scriptPath\n\t- modified syntax\n\t- ...\n...modified syntaxVersion","path":"/ipfs/QmRhQQECYQJABS4nuyCHJaoQr5PbSecFb8vGvBoB4Myfpr","qri":"cm:0","signature":"njCFxpGqq0xJSrjgxC289KncjflqA0e00txweEqIyUTvEKSUBKHcfQmx4OQIJzJqQJdcjIEzFrwP9cdquozRgsnrpsSfKb+wBWdtbnrg8zfat0X/Dqjro6JD7afJf0gU9s5SDi/s8g/qZOLwWh1nuoH4UAeUX+l3DH0ocFjeD6r/YkMJ0KXaWaFloKP8UPasfqoei9PxxmYQuAnFMqpXFisB7mKFAbgbpF3eL80UcbQPTih7WF11SBym/AzJhGNvOivOjmRxKGEuqEH9g3NPTEQr+LnP415X4qiaZA6MVmOO66vC0diUN4vJUMvhTsWnVEBtgqjTRYlSaYwabHv/gA==","timestamp":"2001-01-01T01:02:01.000000001Z","title":"Transform: 3 changes"},"meta":{"qri":"md:0","title":"example movie data"},"path":"/ipfs/QmS1jcwNLc1wYjBVuhYniKEX2RPGv5dppwSfRpcjVbMaHG","peername":"me","previousPath":"/ipfs/QmdxjWGrjc9neXqReY6bHMC4eGG5je358PcCWCHNVYbLGU","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"transform":{"qri":"tf:0","scriptPath":"/ipfs/Qmb69tx5VCL7q7EfkGKpDgESBysmDbohoLvonpbgri48NN","syntax":"starlark","syntaxVersion":"0.9.2-dev"},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmXkN5J5yCAtF8GCxwRXARzAQhj3bPaSv1VHoyCCXzQRzN","scriptPath":"/ipfs/QmVM37PFzBcZn3qqKvyQ9rJ1jC8NkS8kYZNJke1Wje1jor"}}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("dataset (-want +got):\n%s", diff)
	}
}

// Test that save can override just the viz
func TestSaveThenOverrideViz(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_file_transform")
	defer run.Delete()

	// Save a version, then save another with a viz template
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/template.html me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.RepoRoot.GetPathForDataset(0)
	actual := run.RepoRoot.DatasetMarshalJSON(dsPath)

	// This dataset is ds_ten.yaml, with an added viz section
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"\t- modified scriptPath\n","path":"/ipfs/QmZJGdSvx9ejepMiTcgfBHgBMXwC9LhusPuVZjHUn2i4WM","qri":"cm:0","signature":"njCFxpGqq0xJSrjgxC289KncjflqA0e00txweEqIyUTvEKSUBKHcfQmx4OQIJzJqQJdcjIEzFrwP9cdquozRgsnrpsSfKb+wBWdtbnrg8zfat0X/Dqjro6JD7afJf0gU9s5SDi/s8g/qZOLwWh1nuoH4UAeUX+l3DH0ocFjeD6r/YkMJ0KXaWaFloKP8UPasfqoei9PxxmYQuAnFMqpXFisB7mKFAbgbpF3eL80UcbQPTih7WF11SBym/AzJhGNvOivOjmRxKGEuqEH9g3NPTEQr+LnP415X4qiaZA6MVmOO66vC0diUN4vJUMvhTsWnVEBtgqjTRYlSaYwabHv/gA==","timestamp":"2001-01-01T01:02:01.000000001Z","title":"Viz: 1 change"},"meta":{"qri":"md:0","title":"example movie data"},"path":"/ipfs/QmbGRMX6NbJhkV46GZaRpiHfLox4zVEKRbiiyCUNamFsjx","peername":"me","previousPath":"/ipfs/QmdxjWGrjc9neXqReY6bHMC4eGG5je358PcCWCHNVYbLGU","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmVrEH7T7XmdJLym8YL9DjwCALbz264h7GQTrjkSGmbvry","scriptPath":"/ipfs/QmRaVGip3V9fVBJheZN6FbUajD3ZLNjHhXdjrmfg2JPoo5"}}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("dataset (-want +got):\n%s", diff)
	}
}

// Test that save can combine a meta compoent, and a transform, and a viz
func TestSaveThenOverrideMetaAndTransformAndViz(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_file_transform")
	defer run.Delete()

	// Save a version, then save another with three components at once
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/movies/meta_override.yaml --file=testdata/movies/tf.star --file=testdata/template.html me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.RepoRoot.GetPathForDataset(0)
	actual := run.RepoRoot.DatasetMarshalJSON(dsPath)

	// This dataset is ds_ten.yaml, with an added meta component, and transform, and viz
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"\t- modified scriptPath\n\t- modified syntax\n\t- ...\n...modified syntaxVersion","path":"/ipfs/QmRhQQECYQJABS4nuyCHJaoQr5PbSecFb8vGvBoB4Myfpr","qri":"cm:0","signature":"njCFxpGqq0xJSrjgxC289KncjflqA0e00txweEqIyUTvEKSUBKHcfQmx4OQIJzJqQJdcjIEzFrwP9cdquozRgsnrpsSfKb+wBWdtbnrg8zfat0X/Dqjro6JD7afJf0gU9s5SDi/s8g/qZOLwWh1nuoH4UAeUX+l3DH0ocFjeD6r/YkMJ0KXaWaFloKP8UPasfqoei9PxxmYQuAnFMqpXFisB7mKFAbgbpF3eL80UcbQPTih7WF11SBym/AzJhGNvOivOjmRxKGEuqEH9g3NPTEQr+LnP415X4qiaZA6MVmOO66vC0diUN4vJUMvhTsWnVEBtgqjTRYlSaYwabHv/gA==","timestamp":"2001-01-01T01:02:01.000000001Z","title":"Transform: 3 changes"},"meta":{"qri":"md:0","title":"different title"},"path":"/ipfs/QmXu6f5GvujYdFFF1v1JgBw1EbrYCvcwz2mFY8mZkje3DX","peername":"me","previousPath":"/ipfs/QmdxjWGrjc9neXqReY6bHMC4eGG5je358PcCWCHNVYbLGU","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"transform":{"qri":"tf:0","scriptPath":"/ipfs/Qmb69tx5VCL7q7EfkGKpDgESBysmDbohoLvonpbgri48NN","syntax":"starlark","syntaxVersion":"0.9.2-dev"},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmVrEH7T7XmdJLym8YL9DjwCALbz264h7GQTrjkSGmbvry","scriptPath":"/ipfs/QmRaVGip3V9fVBJheZN6FbUajD3ZLNjHhXdjrmfg2JPoo5"}}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("dataset (-want +got):\n%s", diff)
	}
}

// Test that saving a full dataset with a component at the same time is an error
func TestSaveDatasetWithComponentError(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_then_override_meta")
	defer run.Delete()

	// Try to save with two conflicting components, but this returns an error
	err := run.ExecCommand("qri save --file=testdata/movies/ds_ten.yaml --file=testdata/movies/meta_override.yaml me/test_ds")
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

	run := NewTestRunner(t, "test_peer", "qri_test_save_then_override_meta")
	defer run.Delete()

	// Save two versions, but second has a conflict error
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	err := run.ExecCommand("qri save --file=testdata/movies/meta_override.yaml --file=testdata/movies/meta_override.yaml me/test_ds")
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

	run := NewTestRunner(t, "test_peer", "qri_test_transform_same")
	defer run.Delete()

	// Save a version, then another with no changes
	run.MustExec(t, "qri save --file=testdata/movies/tf_123.star me/test_ds")
	err := run.ExecCommand("qri save --file=testdata/movies/tf_123.star me/test_ds")
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

	run := NewTestRunner(t, "test_peer", "qri_test_save_transform_get_body")
	defer run.Delete()

	// Save two versions, the second of which uses get_body in a transformation
	run.MustExec(t, "qri save --body=testdata/movies/body_two.json me/test_ds")
	run.MustExec(t, "qri save --file=testdata/movies/tf_add_one.star me/test_ds")

	// Read body from the dataset that was created with the transform
	dsPath := run.RepoRoot.GetPathForDataset(0)
	actualBody := run.RepoRoot.ReadBodyFromIPFS(dsPath + "/body.json")

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

	run := NewTestRunner(t, "test_peer", "qri_test_diff_revisions")
	defer run.Delete()

	// Save three versions, then diff the last two
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	output := run.MustExec(t, "qri diff body me/test_movies")

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

	run := NewTestRunner(t, "test_peer", "qri_test_diff_only_one")
	defer run.Delete()

	// Save a version, then try to diff but it returns an error because there's only one version
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	err := run.ExecCommand("qri diff body me/test_movies")
	if err == nil {
		t.Errorf("expected error, did not get one")
	}
	expect := `dataset has only one version, nothing to diff against`
	if err.Error() != expect {
		t.Errorf("expected error: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

// Test that save can be called with a readme file
func TestSaveReadmeFromFile(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "save_readme_file")
	defer run.Delete()

	// Save two versions, one with a body, the second with a readme
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/save_readme_file")
	run.MustExec(t, "qri save --file=testdata/movies/about_movies.md me/save_readme_file")

	// Verify we can get the readme back
	actual := run.MustExec(t, "qri get readme me/save_readme_file")
	expect := `format: md
qri: rm:0
scriptPath: /ipfs/QmQPbLdDwyAzCmKayuHGeNGx5eboDv5aXTMuw2daUuneCb

`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("readme.md contents (-want +got):\n%s", diff)
	}

	// As well as the readme script bytes
	actual = run.MustExec(t, "qri get readme.script me/save_readme_file")
	expect = `# Title

This is a dataset about movies

`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("readme.md contents (-want +got):\n%s", diff)
	}
}
