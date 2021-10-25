package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	golog "github.com/ipfs/go-log"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/config"
	repotest "github.com/qri-io/qri/repo/test"
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
	addr, err := ma.NewMultiaddr(config.DefaultAPIAddress)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	l, err := manet.Listen(addr)
	if err != nil {
		return fmt.Errorf("it looks like a qri server is already running on address %s, please close before running tests", config.DefaultAPIAddress)
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

// Test that saving a dataset with a relative body path works, and validate the contents of that
// body match what was given to the save command.
func TestSaveRelativeBodyPath(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer_save_relative_body", "qri_test_save_relative_body")
	defer run.Delete()

	// Save a dataset which has a body as a relative path
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_movies")

	// Read body from the dataset that was saved.
	dsPath := run.GetPathForDataset(t, 0)
	actualBody := run.ReadBodyFromIPFS(t, dsPath+"/body.csv")

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

	run := NewTestRunner(t, "test_peer_remove_only_two_revisions", "qri_test_remove_only_two_revisions")
	defer run.Delete()

	// Save three revisions, then remove two
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	run.MustExec(t, "qri remove me/test_movies --revisions=2")

	// Read body from the dataset that was saved.
	dsPath := run.GetPathForDataset(t, 0)
	actualBody := run.ReadBodyFromIPFS(t, dsPath+"/body.csv")

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

	run := NewTestRunner(t, "test_peer_remove_all_revisions_long_form", "qri_test_remove_all_revisions_long_form")
	defer run.Delete()

	// Save three versions, then remove all of them.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	run.MustExec(t, "qri remove me/test_movies --revisions=all")

	// Read path for dataset, which shouldn't exist anymore.
	dsPath := run.GetPathForDataset(t, 0)
	if dsPath != "" {
		t.Errorf("expected dataset to be removed entirely, found at \"%s\"", dsPath)
	}
}

// Test that adding three revision, then removing all of them leaves nothing, using --all.
func TestRemoveAllRevisionsShortForm(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer_remove_all_revisions_short_form", "qri_test_remove_all_revisions_short_form")
	defer run.Delete()

	// Save three versions, then remove all of them, using the --all flag.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	run.MustExec(t, "qri remove me/test_movies --all")

	// Read path for dataset, which shouldn't exist anymore.
	dsPath := run.GetPathForDataset(t, 0)
	if dsPath != "" {
		t.Errorf("expected dataset to be removed entirely, found at \"%s\"", dsPath)
	}
}

// Test that save can override a single component, meta in this case.
func TestSaveThenOverrideMetaComponent(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer_save_then_override_meta", "qri_test_save_then_override_meta")
	defer run.Delete()

	// Save a version, then save another with a new meta component.
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/movies/meta_override.yaml me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.GetPathForDataset(t, 0)
	got := run.MustLoadDataset(t, dsPath)

	// This dataset is ds_ten.yaml, with the meta replaced by meta_override.yaml.
	dstest.CompareGoldenDatasetAndUpdateIfEnvVarSet(t, "testdata/expect/TestSaveThenOverrideMetaComponent.json", got)
}

// Test save with a body, then adding a meta
func TestSaveWithBodyThenAddMetaComponent(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer_save_with_body_then_override_meta", "qri_test_save_with_body_then_override_meta")
	defer run.Delete()

	// Save a version with a csv body, then another with a new meta component.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/simple_ds")
	run.MustExec(t, "qri save --file=testdata/movies/meta_override.yaml me/simple_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.GetPathForDataset(t, 0)
	got := run.MustLoadDataset(t, dsPath)

	// This version has a commit message about the meta being added
	dstest.CompareGoldenDatasetAndUpdateIfEnvVarSet(t, "testdata/expect/TestSaveWithBodyThenAddMetaComponent.json", got)
}

// Test save with a body, then adding a meta
func TestSaveWithBodyThenAddMetaAndSmallBodyChange(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer_save_then_override_meta_and_body", "qri_test_save_then_override_meta_and_body")
	defer run.Delete()

	// Save a version with a csv body, then another with a new meta component and different body.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/simple_ds")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv --file=testdata/movies/meta_override.yaml me/simple_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.GetPathForDataset(t, 0)
	got := run.MustLoadDataset(t, dsPath)

	// This version has a commit message about the meta being added and body changing
	dstest.CompareGoldenDatasetAndUpdateIfEnvVarSet(t, "testdata/expect/TestSaveWithBodyThenAddMetaAndSmallBodyChange.json", got)
}

// Test that saving with two components at once will merge them together.
func TestSaveTwoComponents(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer_save_two_components", "qri_test_save_two_component")
	defer run.Delete()

	// Save a version, then same another with two components at once
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/movies/meta_override.yaml --file=testdata/movies/structure_override.json me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.GetPathForDataset(t, 0)
	got := run.MustLoadDataset(t, dsPath)

	// This dataset is ds_ten.yaml, with the meta replaced by meta_override ("different title") and
	// the structure replaced by structure_override (lazyQuotes: false && title: "name").
	dstest.CompareGoldenDatasetAndUpdateIfEnvVarSet(t, "testdata/expect/TestSaveTwoComponents.json", got)
}

// Test that save can override just the transform
func TestSaveThenOverrideTransform(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer_save_file_transform", "qri_test_save_file_transform")
	defer run.Delete()

	// Save a version, then save another with a transform
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --apply --file=testdata/movies/tf.star me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.GetPathForDataset(t, 0)
	got := run.MustLoadDataset(t, dsPath)

	// This dataset is ds_ten.yaml, with an added transform section
	dstest.CompareGoldenDatasetAndUpdateIfEnvVarSet(t, "testdata/expect/TestSaveThenOverrideTransform.json", got)
}

// Test that save can override just the viz
func TestSaveThenOverrideViz(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer_save_file_viz", "qri_test_save_file_viz")
	defer run.Delete()

	// Save a version, then save another with a viz template
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/template.html me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.GetPathForDataset(t, 0)
	got := run.MustLoadDataset(t, dsPath)

	// This dataset is ds_ten.yaml, with an added viz section
	dstest.CompareGoldenDatasetAndUpdateIfEnvVarSet(t, "testdata/expect/TestSaveThenOverrideViz.json", got)
}

// Test that save can combine a meta compoent, and a transform, and a viz
func TestSaveThenOverrideMetaAndTransformAndViz(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer_save_then_override_meta_transform_viz", "qri_test_save_then_override_meta_transfrom_viz")
	defer run.Delete()

	// Save a version, then save another with three components at once
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --apply --file=testdata/movies/meta_override.yaml --file=testdata/movies/tf.star --file=testdata/template.html me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.GetPathForDataset(t, 0)
	got := run.MustLoadDataset(t, dsPath)

	// This dataset is ds_ten.yaml, with an added meta component, and transform, and viz
	dstest.CompareGoldenDatasetAndUpdateIfEnvVarSet(t, "testdata/expect/TestSaveThenOverrideMetaAndTransformAndViz.json", got)
}

// Test that saving a full dataset with a component at the same time is an error
func TestSaveDatasetWithComponentError(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer_with_component_error", "qri_test_save_with_component_error")
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

	run := NewTestRunner(t, "test_peer_save_conflicting_components", "qri_test_save_conflicting_components")
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

	run := NewTestRunner(t, "test_peer_transform_without_changes", "qri_test_transform_without_changes")
	defer run.Delete()

	// Save a version, then another with no changes
	run.MustExec(t, "qri save --apply --file=testdata/movies/tf_123.star me/test_ds")

	errOut := run.GetCommandErrOutput()
	if !strings.Contains(errOut, "setting body") {
		t.Errorf("expected ErrOutput to contain print statement from transform script. errOutput:\n%s", errOut)
	}

	err := run.ExecCommand("qri save --apply --file=testdata/movies/tf_123.star me/test_ds")
	expect := `saving failed: no changes`
	if err == nil {
		t.Fatalf("expected error: did not get one")
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

	run := NewTestRunner(t, "test_peer_save_transform_using_get_and_set_body", "qri_test_save_transform_get_and_set_body")
	defer run.Delete()

	// Save two versions, the second of which uses get_body in a transformation
	run.MustExec(t, "qri save --body=testdata/movies/body_two.json me/test_ds")
	run.MustExec(t, "qri save --apply --file=testdata/movies/tf_add_row.star me/test_ds")

	// Read body from the dataset that was created with the transform
	dsPath := run.GetPathForDataset(t, 0)
	actualBody := run.ReadBodyFromIPFS(t, dsPath+"/body.json")

	// This body is body_two.json, with the numbers in the second column increased by 1.
	expectBody := `[["Avatar",178],["Pirates of the Caribbean: At World's End",169],["Batman",126]]`
	if actualBody != expectBody {
		t.Errorf("error, dataset actual:\n%s\nexpect:\n%s\n", actualBody, expectBody)
	}
}

// Test that modifying a transform that produces the same body results in a new version
func TestSaveTransformModifiedButSameBody(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer_transform_modified", "qri_test_transform_modified")
	defer run.Delete()

	// Save a version
	run.MustExec(t, "qri save --apply --file=testdata/movies/tf_123.star me/test_ds")

	// Save another version with a modified transform that produces the same body
	err := run.ExecCommand("qri save --apply --file=testdata/movies/tf_modified.star me/test_ds")

	if err != nil {
		t.Errorf("unexpected error: %q", err)
	}

	output := run.MustExec(t, "qri log me/test_ds")
	expect := dstest.Template(t, `1   Commit:  {{ .commit1 }}
    Date:    Sun Dec 31 20:02:01 EST 2000
    Storage: local
    Size:    6 B

    transform added text
    transform:
    	added text

2   Commit:  {{ .commit2 }}
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: local
    Size:    6 B

    created dataset from tf_123.star

`, map[string]string{
		"commit1": "/ipfs/QmW8WzjmaUTHpjAXJr8NYXaMMU3RCAkGJ5WCMWDGGo4n8p",
		"commit2": "/ipfs/QmYSEZWTzZEAArSN5fUVXAezExhTXa4hxyyzSYsafpXvJR",
	})
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("log (-want +got):\n%s", diff)
	}
}

// Test that save can be called with a readme file
func TestSaveReadmeFromFile(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_readme_file", "qri_test_save_readme_file")
	defer run.Delete()

	// Save two versions, one with a body, the second with a readme
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/save_readme_file")
	run.MustExec(t, "qri save --file=testdata/movies/about_movies.md me/save_readme_file")

	// Verify we can get the readme back
	actual := run.MustExec(t, "qri get readme me/save_readme_file")
	expect := `format: md
qri: rm:0
text: |
  # Title

  This is a dataset about movies

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

// Test that renaming a dataset after registration (which changes the username) works correctly
func TestRenameAfterRegistration(t *testing.T) {
	run := NewTestRunnerWithTempRegistry(t, "test_peer_rename_after_reg", "rename_after_reg")
	defer run.Delete()

	tmplData := map[string]string{
		"profileID": "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
		"path":      "/ipfs/QmVmAAVSVewv6HzojRBr2bqJgWwZ8w18vVPqQ6VuTuH7UZ",
	}

	// Create a dataset, using the "anonymous" generated username.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/first_name")

	// Verify the raw references in the repo
	output := run.MustExec(t, "qri list --raw")
	expect := dstest.Template(t, `0 Peername:  test_peer_rename_after_reg
  ProfileID: {{ .profileID }}
  Name:      first_name
  Path:      {{ .path }}
  Published: false

`, tmplData)

	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Register (using a mock server) which changes the username
	err := run.ExecCommandWithStdin(run.Context, "qri registry signup --username real_peer --email me@example.com", "myPassword")
	if err != nil {
		t.Fatal(err)
	}

	output = run.MustExec(t, "qri list --raw")
	expect = dstest.Template(t, `0 Peername:  real_peer
  ProfileID: {{ .profileID }}
  Name:      first_name
  Path:      {{ .path }}
  Published: false

`, tmplData)
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Rename the created dataset, which should work even though our username changed
	run.MustExec(t, "qri rename me/first_name me/second_name")

	output = run.MustExec(t, "qri list --raw")
	expect = dstest.Template(t, `0 Peername:  real_peer
  ProfileID: {{ .profileID }}
  Name:      second_name
  Path:      {{ .path }}
  Published: false

`, tmplData)

	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Rename a second time, make sure this works still
	run.MustExec(t, "qri rename me/second_name me/third_name")

	output = run.MustExec(t, "qri list --raw")
	expect = dstest.Template(t, `0 Peername:  real_peer
  ProfileID: {{ .profileID }}
  Name:      third_name
  Path:      {{ .path }}
  Published: false

`, tmplData)
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}

// Test that list can format output as json
func TestListFormatJson(t *testing.T) {
	run := NewTestRunner(t, "test_peer_list_format_json", "list_format_json")
	defer run.Delete()

	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/my_ds")

	// Verify the references in json format
	output := run.MustExec(t, "qri list --format json")
	expect := dstest.Template(t, `[
  {
    "initID": "ph45s2teqrzwlsvb5fjkkd5zt4wkenqspdwdna6zrarwi6jpo7ea",
    "username": "test_peer_list_format_json",
    "profileID": "{{ .profileID }}",
    "name": "my_ds",
    "path": "{{ .path }}",
    "bodySize": 224,
    "bodyRows": 8,
    "bodyFormat": "csv",
    "numErrors": 1,
    "commitTime": "2001-01-01T01:01:01.000000001Z",
    "commitTitle": "created dataset from body_ten.csv",
    "commitMessage": "created dataset from body_ten.csv",
    "commitCount": 1
  }
]`, map[string]string{
		"profileID": "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
		"path":      "/ipfs/QmVmAAVSVewv6HzojRBr2bqJgWwZ8w18vVPqQ6VuTuH7UZ",
	})

	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}

// Test that a dataset name with bad upper-case characters, even if it already exists,
// produces an error and needs to be renamed
func TestBadCaseIsAnError(t *testing.T) {
	run := NewTestRunner(t, "test_peer_qri_get_bad_case", "qri_get_bad_case")
	defer run.Delete()

	// Construct a dataset in order to have an existing version in the repo.
	ds := dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[[\"one\",2],[\"three\",4]]")))

	// Add the dataset to the repo directly, which avoids the name validation check.
	run.AddDatasetToRefstore(t, "test_peer_qri_get_bad_case/a_New_Dataset", &ds)

	// Save the dataset, get an error because it needs to be renamed
	err := run.ExecCommand("qri get test_peer_qri_get_bad_case/a_New_Dataset")
	if err == nil {
		t.Fatalf("expected an error, didn't get one")
	}
	expectErr := `"test_peer_qri_get_bad_case/a_New_Dataset" is not a valid dataset reference: dataset name may not contain any upper-case letters`
	if expectErr != err.Error() {
		t.Errorf("error mismatch, expect: %q, got: %q", expectErr, err)
	}
}

func TestSetupHappensBeforeOtherCommands(t *testing.T) {
	ctx := context.Background()

	qriHome := createTmpQriHome(t)
	cmd, shutdown := newCommand(ctx, qriHome, repotest.NewTestCrypto())

	cmdTextList := []string{"qri connect", "qri diff", "qri get", "qri list", "qri pull", "qri search"}
	expect := "no qri repo exists\nhave you run 'qri setup'?"

	for _, cmdText := range cmdTextList {
		noRepoErr := executeCommand(cmd, cmdText)
		if noRepoErr.Error() != expect {
			timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
			err := fmt.Sprintf("expected error for command %v:\n %v\n but received error:\n %v", cmdText, expect, noRepoErr)
			t.Fatal(err)
		}
	}
}

func parseRefFromSave(output string) string {
	pos := strings.Index(output, "saved: ")
	if pos == -1 {
		panic(fmt.Errorf("expected output to contain \"saved:\", got %q", output))
	}
	ref := output[pos+7:]
	endPos := strings.Index(ref, "\n")
	if endPos > -1 {
		ref = ref[:endPos]
	}
	return strings.TrimSpace(ref)
}
