package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dscache"
	qrierr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/event"
)

func TestSaveComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_complete", "qri_test_save_complete")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args   []string
		expect string
		err    string
	}{
		{[]string{}, "", ""},
		{[]string{"test"}, "test", ""},
	}

	for i, c := range cases {
		opt := &SaveOptions{
			IOStreams: run.Streams,
		}

		opt.Complete(f, c.args)

		if c.err != run.ErrStream.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, run.ErrStream.String())
			run.IOReset()
			continue
		}

		if c.expect != opt.Refs.Ref() {
			t.Errorf("case %d, opt.Ref not set correctly. Expected: '%s', Got: '%s'", i, c.expect, opt.Refs.Ref())
			run.IOReset()
			continue
		}

		if opt.DatasetMethods == nil {
			t.Errorf("case %d, opt.DatasetMethods not set.", i)
			run.IOReset()
			continue
		}
		run.IOReset()
	}
}

func TestSaveValidate(t *testing.T) {
	cases := []struct {
		ref      string
		filepath string
		bodypath string
		err      string
		msg      string
	}{
		{"me/test", "test/path.yaml", "", "", ""},
		{"me/test", "", "test/bodypath.yaml", "", ""},
		{"me/test", "test/filepath.yaml", "test/bodypath.yaml", "", ""},
	}
	for i, c := range cases {
		opt := &SaveOptions{
			Refs:      NewExplicitRefSelect(c.ref),
			FilePaths: []string{c.filepath},
			BodyPath:  c.bodypath,
		}

		err := opt.Validate()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%s'", i, c.err, err)
			continue
		}

		if libErr, ok := err.(qrierr.Error); ok {
			if libErr.Message() != c.msg {
				t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: '%s'", i, c.msg, libErr.Message())
				continue
			}
		} else if c.msg != "" {
			t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: ''", i, c.msg)
			continue
		}
	}
}

func TestSaveRun(t *testing.T) {
	run := NewTestRunner(t, "peer_name", "qri_test_save_run")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		description string
		ref         string
		filepath    string
		bodypath    string
		title       string
		message     string
		dryrun      bool
		noRender    bool
		expect      string
		err         string
		msg         string
	}{
		{"no data", "me/bad_dataset", "", "", "", "", false, true, "", "no changes to save", ""},
		{"bad dataset file", "me/cities", "bad/filpath.json", "", "", "", false, true, "", "open bad/filpath.json: no such file or directory", ""},
		{"bad body file", "me/cities", "", "bad/bodypath.csv", "", "", false, true, "", "opening dataset.bodyPath 'bad/bodypath.csv': path not found", ""},
		{"good inputs, dryrun", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_ten.csv", "", "", true, true, "dataset saved: peer/movies\n", "", ""},
		{"good inputs", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_ten.csv", "", "", false, true, "dataset saved: peer/movies@/mem/QmbCGpS2b1LisRS6D3Q9V7zvtY8jP5arsW3GubrtFHHW5F\nthis dataset has 1 validation errors\n", "", ""},
		{"add rows, dry run", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_twenty.csv", "Added 10 more rows", "Adding to the number of rows in dataset", true, true, "dataset saved: peer/movies\n", "", ""},
		{"add rows, save", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_twenty.csv", "Added 10 more rows", "Adding to the number of rows in dataset", false, true, "dataset saved: peer/movies@/mem/QmebZ1FggGXqbNTkvsLC9jiehCGL3PHewdVszdg3HoFv7a\nthis dataset has 1 validation errors\n", "", ""},
		{"no changes", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_twenty.csv", "trying to add again", "hopefully this errors", false, true, "", "error saving: no changes", ""},
		{"add viz", "me/movies", "testdata/movies/dataset_with_viz.json", "", "", "", false, false, "dataset saved: peer/movies@/mem/QmZXRSEtg8afx47ZQQLV2ec4c7RjG3grYZGvgz1ozq1V5p\nthis dataset has 1 validation errors\n", "", ""},
	}

	for _, c := range cases {
		run.IOReset()
		dsm, err := f.DatasetMethods()
		if err != nil {
			t.Errorf("case \"%s\", error creating dataset request: %s", c.description, err)
			continue
		}

		pathList := []string{}
		if c.filepath != "" {
			pathList = []string{c.filepath}
		}

		opt := &SaveOptions{
			IOStreams:      run.Streams,
			Refs:           NewExplicitRefSelect(c.ref),
			FilePaths:      pathList,
			BodyPath:       c.bodypath,
			Title:          c.title,
			Message:        c.message,
			DryRun:         c.dryrun,
			NoRender:       c.noRender,
			DatasetMethods: dsm,
		}

		err = opt.Run()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case '%s', mismatched error. Expected: '%s', Got: '%v'", c.description, c.err, err)
			continue
		}

		if libErr, ok := err.(qrierr.Error); ok {
			if libErr.Message() != c.msg {
				t.Errorf("case '%s', mismatched user-friendly message. Expected: '%s', Got: '%s'", c.description, c.msg, libErr.Message())
				continue
			}
		} else if c.msg != "" {
			t.Errorf("case '%s', mismatched user-friendly message. Expected: '%s', Got: ''", c.description, c.msg)
			continue
		}

		if c.expect != run.ErrStream.String() {
			t.Errorf("case '%s', err output mismatch. Expected: '%s', Got: '%s'", c.description, c.expect, run.ErrStream.String())

			continue
		}
	}
}

func TestSaveState(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_state", "qri_test_save_state")
	defer run.Delete()

	// Save a csv file
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv test_peer_save_state/my_ds")

	// Read dataset from IPFS and compare it to the expected value
	dsPath := run.GetPathForDataset(t, 0)
	gotDs := run.MustLoadDataset(t, dsPath)
	expectDs := dstest.LoadGoldenFile(t, "testdata/expect/TestSaveState.json")
	if diff := dstest.CompareDatasets(expectDs, gotDs); diff != "" {
		t.Errorf("dataset (-want +got):\n%s", diff)
		dstest.UpdateGoldenFileIfEnvVarSet("testdata/expect/TestSaveState.json", gotDs)
	}

	// Read data and compare it
	actual := run.ReadBodyFromIPFS(t, dsPath+"/body.csv")
	expect := `movie_title,duration
Avatar ,178
Pirates of the Caribbean: At World's End ,169
Spectre ,148
The Dark Knight Rises ,164
Star Wars: Episode VII - The Force Awakens             ,
John Carter ,132
Spider-Man 3 ,156
Tangled ,100
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("body (-want +got):\n%s", diff)
	}

	// Check the log matches what is expected
	actual = run.MustExec(t, "qri log test_peer_save_state/my_ds")
	expect = `1   Commit:  /ipfs/Qmb6HLvyhZe768NLRjxDUe3zMA75yWiex7Kwq86pipQGBf
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: local
    Size:    224 B

    created dataset from body_ten.csv

`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("log (-want +got):\n%s", diff)
	}
}

func TestSaveBasicCommands(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.Chdir(pwd)
		os.RemoveAll(tmpPath)
	}()

	// Copy some files into tmpPath, change to it
	copyFile(t, "testdata/movies/ds_ten.yaml", filepath.Join(tmpPath, "dataset.yaml"))
	copyFile(t, "testdata/movies/body_ten.csv", filepath.Join(tmpPath, "body_ten.csv"))
	copyFile(t, "testdata/movies/structure_override.json", filepath.Join(tmpPath, "structure.json"))
	os.Chdir(tmpPath)

	goodCases := []struct {
		description string
		command     string
		expect      string
	}{
		{
			"dataset file infer name",
			"qri save --file dataset.yaml",
			"dataset saved: test_peer_save_basic/ten_movies@/ipfs/QmTHLYTXR5AM6CoTyAfh3Fu2L9pN4hphR21QxciTh7pXgA\nthis dataset has 1 validation errors\n",
		},
		{
			"dataset file me ref",
			"qri save --file dataset.yaml me/my_dataset",
			"dataset saved: test_peer_save_basic/my_dataset@/ipfs/QmTHLYTXR5AM6CoTyAfh3Fu2L9pN4hphR21QxciTh7pXgA\nthis dataset has 1 validation errors\n",
		},
		{
			"dataset file explicit ref",
			"qri save --file dataset.yaml test_peer_save_basic/my_dataset",
			"dataset saved: test_peer_save_basic/my_dataset@/ipfs/QmTHLYTXR5AM6CoTyAfh3Fu2L9pN4hphR21QxciTh7pXgA\nthis dataset has 1 validation errors\n",
		},
		{
			"body file infer name",
			"qri save --body body_ten.csv",
			"dataset saved: test_peer_save_basic/body_ten@/ipfs/Qmb6HLvyhZe768NLRjxDUe3zMA75yWiex7Kwq86pipQGBf\nthis dataset has 1 validation errors\n",
		},
		{
			"body file me ref",
			"qri save --body body_ten.csv me/my_dataset",
			"dataset saved: test_peer_save_basic/my_dataset@/ipfs/Qmb6HLvyhZe768NLRjxDUe3zMA75yWiex7Kwq86pipQGBf\nthis dataset has 1 validation errors\n",
		},
		{
			"body file explicit ref",
			"qri save --body body_ten.csv test_peer_save_basic/my_dataset",
			"dataset saved: test_peer_save_basic/my_dataset@/ipfs/Qmb6HLvyhZe768NLRjxDUe3zMA75yWiex7Kwq86pipQGBf\nthis dataset has 1 validation errors\n",
		},

		// TODO(dustmop): It's intended that a user can save a dataset with a structure but no
		// body. At some point that functionality broke, because there was no test for it. Fix that
		// in a follow-up change.
		//{
		//	"structure file me ref",
		//	"qri save --file structure.json me/my_dataset",
		//	"TODO(dustmop): Should be possible to save a dataset with structure and no body",
		//},
		//{
		//	"structure file explicit ref",
		//	"qri save --file structure.json test_peer_save_basic/my_dataset",
		//	"TODO(dustmop): Should be possible to save a dataset with structure and no body",
		//},
	}
	for _, c := range goodCases {
		t.Run(c.description, func(t *testing.T) {
			// TODO(dustmop): Would be preferable to instead have a way to clear the refstore
			run := NewTestRunner(t, "test_peer_save_basic", "qri_test_save_basic")
			defer run.Delete()

			err := run.ExecCommandCombinedOutErr(c.command)
			if err != nil {
				t.Errorf("error %s\n", err)
				return
			}
			actual := parseDatasetRefFromOutput(run.GetCommandOutput())
			// vi := run.LookupVersionInfo(t, "test_peer_save_basic/ten_movies")
			// ds := run.MustLoadDataset(t, vi.Path)
			// data, _ := json.MarshalIndent(ds, "", "  ")
			// fmt.Println(string(data))
			if diff := cmp.Diff(c.expect, actual); diff != "" {
				t.Errorf("result mismatch (-want +got):%s\n", diff)
			}
		})
	}

	badCases := []struct {
		description string
		command     string
		expectErr   string
	}{
		{
			"dataset file other username",
			"qri save --file dataset.yaml other/my_dataset",
			"cannot save using a different username than \"qri_test_save_basic_bad_cases\"",
		},
		{
			"dataset file explicit version",
			"qri save --file dataset.yaml me/my_dataset@/ipfs/QmVersion",
			"unexpected character '@', ref can only have username/name",
		},
		{
			"dataset file bad parse",
			"qri save --file dataset.yaml me/invalid+name",
			"dataset name must start with a lower-case letter, and only contain lower-case letters, numbers, dashes, and underscore. Maximum length is 144 characters",
		},
		{
			"body file other username",
			"qri save --body body_ten.csv other/my_dataset",
			"cannot save using a different username than \"qri_test_save_basic_bad_cases\"",
		},
	}
	for _, c := range badCases {
		t.Run(c.description, func(t *testing.T) {
			run := NewTestRunner(t, "qri_test_save_basic_bad_cases", "qri_test_save_basic_bad_cases")
			defer run.Delete()

			err := run.ExecCommand(c.command)
			if err == nil {
				output := run.GetCommandOutput()
				t.Errorf("expected an error, did not get one, output: %s\n", output)
				return
			}
			if err.Error() != c.expectErr {
				t.Errorf("mismatch, expect: %s, got: %s\n", c.expectErr, err.Error())
			}
		})
	}
}

func TestSaveInferName(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_infer_name", "qri_test_save_infer_name")
	defer run.Delete()

	// Save a dataset with an inferred name.
	output := run.MustExecCombinedOutErr(t, "qri save --body testdata/movies/body_four.json")
	actual := parseDatasetRefFromOutput(output)
	expect := "dataset saved: test_peer_save_infer_name/body_four@/ipfs/QmXpkLdjXjJKhy8H724Fw17vSX3dxRozMmRbn8q2cHqfQD\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save again, get an error because the inferred name already exists.
	err := run.ExecCommand("qri save --body testdata/movies/body_four.json")
	expectErr := `inferred dataset name already exists. To add a new commit to this dataset, run save again with the dataset reference "test_peer_save_infer_name/body_four". To create a new dataset, use --new flag`
	if err == nil {
		t.Errorf("error expected, did not get one")
	}
	if diff := cmp.Diff(expectErr, errorMessage(err)); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save but ensure a new dataset is created.
	output = run.MustExecCombinedOutErr(t, "qri save --body testdata/movies/body_four.json --new")
	actual = parseDatasetRefFromOutput(output)
	expect = "dataset saved: test_peer_save_infer_name/body_four_2@/ipfs/QmPJ8xgWg4JPKkoYPCRTjekUDrB3kTd27dZmjjbCBdhaZZ\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save once again.
	output = run.MustExecCombinedOutErr(t, "qri save --body testdata/movies/body_four.json --new")
	actual = parseDatasetRefFromOutput(output)
	expect = "dataset saved: test_peer_save_infer_name/body_four_3@/ipfs/Qmb68LwEXoznDVn6m4o6SUevb7pptk3CHtpAZxzuCBhSFe\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save a dataset whose body filename starts with a number
	output = run.MustExecCombinedOutErr(t, "qri save --body testdata/movies/2018_winners.csv")
	actual = parseDatasetRefFromOutput(output)
	expect = "dataset saved: test_peer_save_infer_name/dataset_2018_winners@/ipfs/QmQbMdwt6fJno3rZ4pvU1ChjUib6X6o3F2XQwLx9AX13Q9\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save a dataset whose body filename is non-alphabetic
	output = run.MustExecCombinedOutErr(t, "qri save --body testdata/2015-09-16--2016-09-30.csv")
	actual = parseDatasetRefFromOutput(output)
	expect = "dataset saved: test_peer_save_infer_name/dataset_2015-09-16--2016-09-30@/ipfs/QmYyppCGk3GVceYhFDFWvbGMMg487zNXnoMhyYEoGssHfy\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save using a CamelCased body filename
	output = run.MustExecCombinedOutErr(t, "qri save --body testdata/movies/TenMoviesAndLengths.csv")
	actual = parseDatasetRefFromOutput(output)
	expect = "dataset saved: test_peer_save_infer_name/ten_movies_and_lengths@/ipfs/QmSpfQzwaVvyKiQaoAYiLJScqozPXx1pDCsMAqSzuoriRu\nthis dataset has 1 validation errors\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save using a body filename that contains unicode
	output = run.MustExecCombinedOutErr(t, "qri save --body testdata/movies/pira\u00f1a_data.csv")
	actual = parseDatasetRefFromOutput(output)
	expect = "dataset saved: test_peer_save_infer_name/pirana_data@/ipfs/QmfCxsLWZfGTEQAnwwnkarGfRu68VW9nMPbyLfrdVjmxqy\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveFilenameUsedForCommitMessage(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_commit", "qri_test_save_commit")
	defer run.Delete()

	// Save a dataset with a bodyfile.
	output := run.MustExecCombinedOutErr(t, "qri save --body testdata/movies/body_four.json")
	ref := parseRefFromSave(output)

	expect := "created dataset from body_four.json\n\n"

	output = run.MustExec(t, fmt.Sprintf("qri get commit.message %s", ref))
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	output = run.MustExec(t, fmt.Sprintf("qri get commit.title %s", ref))
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save a dataset with a dataset file.
	output = run.MustExecCombinedOutErr(t, "qri save --file testdata/movies/ds_ten.yaml")
	ref = parseRefFromSave(output)

	expect = "created dataset from ds_ten.yaml\n\n"

	output = run.MustExec(t, fmt.Sprintf("qri get commit.message %s", ref))
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	output = run.MustExec(t, fmt.Sprintf("qri get commit.title %s", ref))
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveDrop(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_drop", "qri_test_save_drop")
	defer run.Delete()

	run.MustExec(t, "qri save --body testdata/movies/body_two.json me/drop_stuff")
	run.MustExec(t, "qri save --file testdata/movies/meta_override.yaml me/drop_stuff")
	run.MustExec(t, "qri save --drop md me/drop_stuff")

	// Check that the meta is gone.
	output := run.MustExec(t, "qri get meta me/drop_stuff")
	expect := "null\n\n"
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveFilenameMeta(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_filename_meta", "qri_test_save_filename_meta")
	defer run.Delete()

	// Save a dataset with a bodyfile.
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv me/my_ds")

	// Save to add meta. This meta.json file does not have a "md:0" key, but the filename is name
	// to understand that it is a meta component.
	run.MustExec(t, "qri save --file testdata/detect/meta.json me/my_ds")

	expect := "This is dataset title\n\n"
	output := run.MustExec(t, fmt.Sprintf("qri get meta.title me/my_ds"))
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveDscacheFirstCommit(t *testing.T) {
	run := NewTestRunner(t, "test_peer_dscache_first", "qri_test_dscache_first")
	defer run.Delete()

	// Save a dataset with one version.
	run.MustExec(t, "qri save --body testdata/movies/body_two.json me/movie_ds --use-dscache")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Access the dscache
	r, err := run.RepoRoot.Repo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cache := r.Dscache()

	// Dscache should have one reference. It has topIndex 1 because there are two logbook
	// elements in the branch, one for "init", one for "commit".
	actual := cache.VerboseString(false)
	expect := `Dscache:
 Dscache.Users:
  0) user=test_peer_dscache_first profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = m3wkm4dsizgba52qn6ais7lnn5cz67z5e7ztinj3znn4v4kd3k3a
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = movie_ds
     bodySize      = 79
     bodyRows      = 2
     commitTime    = 978310861
     headRef       = /ipfs/QmT4Li2EW4QAxKGU9dpqE1sqj9sM18d8hpfWjTJxwqSwgD
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	shutdownRepoGraceful(cancel, r)

	// Save a different dataset, but dscache already exists.
	run.MustExec(t, "qri save --body testdata/movies/body_four.json me/another_ds --use-dscache")

	// Because this test is using a memrepo, but the command runner instantiates its own repo
	// the dscache is not reloaded. Manually reload it here by constructing a dscache from the
	// same filename.
	fs, err := localfs.NewFS(nil)
	if err != nil {
		t.Errorf("error creating local filesystem: %s", err)
		return
	}

	cacheFilename := cache.Filename
	cache = dscache.NewDscache(ctx, fs, event.NilBus, run.Username(), cacheFilename)

	// Dscache should have two entries now. They are alphabetized by pretty name, and have all
	// the expected data.
	actual = cache.VerboseString(false)
	expect = `Dscache:
 Dscache.Users:
  0) user=test_peer_dscache_first profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = 5m7avnilayu76zrvhfec6vw6loyvdfs622eagqvfioikqxte5paa
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = another_ds
     bodySize      = 137
     bodyRows      = 4
     commitTime    = 978310921
     headRef       = /ipfs/QmPJ8xgWg4JPKkoYPCRTjekUDrB3kTd27dZmjjbCBdhaZZ
  1) initID        = m3wkm4dsizgba52qn6ais7lnn5cz67z5e7ztinj3znn4v4kd3k3a
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = movie_ds
     bodySize      = 79
     bodyRows      = 2
     commitTime    = 978310861
     headRef       = /ipfs/QmT4Li2EW4QAxKGU9dpqE1sqj9sM18d8hpfWjTJxwqSwgD
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveDscacheExistingDataset(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_dscache_existing_dataset", "qri_test_save_dscache_existing_dataset")
	defer run.Delete()

	// Save a dataset with one version.
	run.MustExec(t, "qri save --body testdata/movies/body_two.json me/movie_ds")

	// List with the --use-dscache flag, which builds the dscache from the logbook.
	run.MustExec(t, "qri list --use-dscache")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Access the dscache
	r, err := run.RepoRoot.Repo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cache := r.Dscache()

	// Dscache should have one reference. It has topIndex 1 because there are two logbook
	// elements in the branch, one for "init", one for "commit".
	actual := cache.VerboseString(false)
	expect := `Dscache:
 Dscache.Users:
  0) user=test_peer_save_dscache_existing_dataset profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = 47gqzvfhitq4prj4omvbjl3cjfta4wt7ywno4dc6e5ad5gp3uopq
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = movie_ds
     bodySize      = 79
     bodyRows      = 2
     commitTime    = 978310861
     headRef       = /ipfs/QmT4Li2EW4QAxKGU9dpqE1sqj9sM18d8hpfWjTJxwqSwgD
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	shutdownRepoGraceful(cancel, r)

	// Save a new new commit. Since the dscache exists, it should get updated.
	run.MustExec(t, "qri save --body testdata/movies/body_four.json me/movie_ds")

	// Because this test is using a memrepo, but the command runner instantiates its own repo
	// the dscache is not reloaded. Manually reload it here by constructing a dscache from the
	// same filename.
	fs, err := localfs.NewFS(nil)
	if err != nil {
		t.Errorf("error creating local filesystem: %s", err)
		return
	}

	cacheFilename := cache.Filename
	cache = dscache.NewDscache(ctx, fs, event.NilBus, run.Username(), cacheFilename)

	// Dscache should now have one reference. Now topIndex is 2 because there is another "commit".
	actual = cache.VerboseString(false)
	expect = `Dscache:
 Dscache.Users:
  0) user=test_peer_save_dscache_existing_dataset profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = 47gqzvfhitq4prj4omvbjl3cjfta4wt7ywno4dc6e5ad5gp3uopq
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 2
     cursorIndex   = 2
     prettyName    = movie_ds
     bodySize      = 137
     bodyRows      = 4
     commitTime    = 978310921
     headRef       = /ipfs/QmeBJ8tPCzuo4unh82BVssqTcTFEU6TZxB2z7W14KK8H9w
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveDscacheThenRemoveAll(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_dscache_remove_all", "qri_test_save_dscache_remove_all")
	defer run.Delete()

	// Save a dataset with one version.
	run.MustExec(t, "qri save --body testdata/movies/body_two.json me/movie_ds")

	// Save another dataset.
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv me/another_ds")

	// List with the --use-dscache flag, which builds the dscache from the logbook.
	run.MustExec(t, "qri list --use-dscache")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Access the dscache
	r, err := run.RepoRoot.Repo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cache := r.Dscache()

	// Dscache should have two references, one for each save operation.
	actual := cache.VerboseString(false)
	expect := `Dscache:
 Dscache.Users:
  0) user=test_peer_save_dscache_remove_all profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = g55pf2kd46giewayohfi5qtyxn4evqf3btsupbjopzukmajqfbla
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = another_ds
     bodySize      = 224
     bodyRows      = 8
     commitTime    = 978310921
     numErrors     = 1
     headRef       = /ipfs/QmbHTGontchCxSetpSKS1Ewat9U954My5SSQM2gaEwe5mT
  1) initID        = otbpnqlrxi2fkb7spfhuys7owxf3nq4k7v7kc2wj725nq4ouccwa
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = movie_ds
     bodySize      = 79
     bodyRows      = 2
     commitTime    = 978310861
     headRef       = /ipfs/QmT4Li2EW4QAxKGU9dpqE1sqj9sM18d8hpfWjTJxwqSwgD
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	shutdownRepoGraceful(cancel, r)

	// Remove one of those datasets.
	run.MustExec(t, "qri remove --all me/another_ds")

	// Because this test is using a memrepo, but the command runner instantiates its own repo
	// the dscache is not reloaded. Manually reload it here by constructing a dscache from the
	// same filename.
	fs, err := localfs.NewFS(nil)
	if err != nil {
		t.Errorf("error creating local filesystem")
		return
	}
	cacheFilename := cache.Filename
	cache = dscache.NewDscache(ctx, fs, event.NilBus, run.Username(), cacheFilename)

	// Dscache should now have one reference.
	actual = cache.VerboseString(false)
	expect = `Dscache:
 Dscache.Users:
  0) user=test_peer_save_dscache_remove_all profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = otbpnqlrxi2fkb7spfhuys7owxf3nq4k7v7kc2wj725nq4ouccwa
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = movie_ds
     bodySize      = 79
     bodyRows      = 2
     commitTime    = 978310861
     headRef       = /ipfs/QmT4Li2EW4QAxKGU9dpqE1sqj9sM18d8hpfWjTJxwqSwgD
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveDscacheThenRemoveVersions(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_dscache_remove", "qri_test_save_dscache_remove")
	defer run.Delete()

	// Save a dataset with one version.
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv me/movie_ds")

	// Save another version.
	run.MustExec(t, "qri save --body testdata/movies/body_twenty.csv me/movie_ds")

	// Save yet another version.
	run.MustExec(t, "qri save --body testdata/movies/body_thirty.csv me/movie_ds")

	// List with the --use-dscache flag, which builds the dscache from the logbook.
	run.MustExec(t, "qri list --use-dscache")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Access the dscache
	r, err := run.RepoRoot.Repo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cache := r.Dscache()

	// Dscache should have one reference. It has three commits.
	actual := cache.VerboseString(false)
	expect := `Dscache:
 Dscache.Users:
  0) user=test_peer_save_dscache_remove profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = lht43zmspwsj3oukq3ivqz6m3yro5qc7wmckmxbgugy3oqrfdgsa
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 3
     cursorIndex   = 3
     prettyName    = movie_ds
     bodySize      = 720
     bodyRows      = 28
     commitTime    = 978310981
     numErrors     = 1
     headRef       = /ipfs/QmQYdpXHwNYNPFQ9mKygqAMKhi5qEacXU6rx1of2m1zzqt
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	shutdownRepoGraceful(cancel, r)

	// Remove one of those commits, keeping 1.
	run.MustExec(t, "qri remove --revisions=1 me/movie_ds")

	// Because this test is using a memrepo, but the command runner instantiates its own repo
	// the dscache is not reloaded. Manually reload it here by constructing a dscache from the
	// same filename.
	fs, err := localfs.NewFS(nil)
	if err != nil {
		t.Errorf("error creating local filesystem: %s", err)
	}
	cacheFilename := cache.Filename
	cache = dscache.NewDscache(ctx, fs, event.NilBus, run.Username(), cacheFilename)

	// Dscache should now have one reference.
	actual = cache.VerboseString(false)
	expect = `Dscache:
 Dscache.Users:
  0) user=test_peer_save_dscache_remove profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = lht43zmspwsj3oukq3ivqz6m3yro5qc7wmckmxbgugy3oqrfdgsa
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 2
     cursorIndex   = 2
     prettyName    = movie_ds
     bodySize      = 224
     bodyRows      = 28
     commitTime    = 978310861
     numErrors     = 1
     headRef       = /ipfs/Qmb6HLvyhZe768NLRjxDUe3zMA75yWiex7Kwq86pipQGBf
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveBadCaseCantBeUsedForNewDatasets(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_bad_case", "qri_save_bad_case")
	defer run.Delete()

	// Try to save a new dataset, but its name has upper-case characters.
	err := run.ExecCommand("qri save --body testdata/movies/body_two.json test_peer_save_bad_case/a_New_Dataset")
	if err == nil {
		t.Fatal("expected error trying to save, did not get an error")
	}
	expect := `dataset name may not contain any upper-case letters`
	if err.Error() != expect {
		t.Errorf("error mismatch, expect: %s, got: %s", expect, err.Error())
	}

	// Construct a dataset in order to have an existing version in the repo.
	ds := dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[[\"one\",2],[\"three\",4]]")))

	// Add the dataset to the repo directly, which avoids the name validation check.
	run.AddDatasetToRefstore(t, "test_peer_save_bad_case/a_New_Dataset", &ds)

	// Save the dataset, which will work now that a version already exists.
	run.MustExec(t, "qri save --body testdata/movies/body_two.json test_peer_save_bad_case/a_New_Dataset")
}

func parseDatasetRefFromOutput(text string) string {
	pos := strings.Index(text, "dataset saved:")
	if pos == -1 {
		return ""
	}
	return text[pos:]
}

func copyFile(t *testing.T, source, destin string) {
	data, err := ioutil.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(destin, data, 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSaveLargeBodyIsSame(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_large_body", "qri_test_save_large_body")
	defer run.Delete()

	prevBodySizeLimit := dsfs.BodySizeSmallEnoughToDiff
	defer func() { dsfs.BodySizeSmallEnoughToDiff = prevBodySizeLimit }()
	dsfs.BodySizeSmallEnoughToDiff = 100

	// Save a first version
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv test_peer_save_large_body/my_ds")

	// Try to save another, but no changes
	err := run.ExecCommand("qri save --body testdata/movies/body_ten.csv test_peer_save_large_body/my_ds")
	if err == nil {
		t.Fatal("expected error trying to save, did not get an error")
	}
	expect := `error saving: no changes`
	if err.Error() != expect {
		t.Errorf("error mismatch, expect: %s, got: %s", expect, err.Error())
	}

	// Save a second version by making changes
	run.MustExec(t, "qri save --body testdata/movies/body_twenty.csv test_peer_save_large_body/my_ds")

	output := run.MustExec(t, "qri log test_peer_save_large_body/my_ds")
	expect = `1   Commit:  /ipfs/QmX1dBZv8SM8fuAbHE4K5usH1v4aFuZEfi226QDpiNJbjj
    Date:    Sun Dec 31 20:02:01 EST 2000
    Storage: local
    Size:    532 B

    body changed

2   Commit:  /ipfs/QmNX9ZKXtdskpYSQ5spd1qvqB2CPoWfJbdAcWoFndintrF
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: local
    Size:    224 B

    created dataset from body_ten.csv

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveTwiceWithTransform(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_twice_with_xform", "qri_test_save_twice_with_xform")
	defer run.Delete()

	// Save a first version with a normal body
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv test_peer_save_twice_with_xform/my_ds")

	// Save a second version with a transform
	run.MustExec(t, "qri save --file testdata/movies/tf_one_movie.star test_peer_save_twice_with_xform/my_ds")

	// Get the saved transform, make sure it matches the source file
	output := run.MustExec(t, "qri get transform.script test_peer_save_twice_with_xform/my_ds")
	golden, _ := ioutil.ReadFile("testdata/movies/tf_one_movie.star")
	expect := strings.TrimSpace(string(golden))
	actual := strings.TrimSpace(output)
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveTransformUsingPrev(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_using_prev", "qri_test_save_using_prev")
	defer run.Delete()

	// Save a first version with a normal body
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv test_peer_save_using_prev/my_ds")

	// Save a second version with a transform
	run.MustExec(t, "qri save --file testdata/movies/tf_set_len.star test_peer_save_using_prev/my_ds")

	// Read body from the dataset that was saved.
	dsPath := run.GetPathForDataset(t, 0)
	actualBody := run.ReadBodyFromIPFS(t, dsPath+"/body.csv")

	// Read the body from the testdata input file.
	expectBody := "movie_title,duration\nNumber of Movies,8\n"

	// Make sure they match.
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveTransformUsingConfigSecret(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_twice_with_xform", "qri_test_save_twice_with_xform")
	defer run.Delete()

	// Save a version with a transform that has config and secret data
	run.MustExec(t, "qri save --file testdata/movies/tf_using_config_secret.json --secrets animal_sound,meow test_peer_save_twice_with_xform/my_ds")

	// Read body from the dataset that was saved.
	dsPath := run.GetPathForDataset(t, 0)
	actualBody := run.ReadBodyFromIPFS(t, dsPath+"/body.json")

	// Expected result has the config and secret data
	expectBody := `[["Name","cat"],["Sound","meow"]]`

	// Make sure they match.
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveTransformSetMeta(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_set_meta", "qri_test_save_set_meta")
	defer run.Delete()

	// Save a first version with a normal body
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv test_peer_save_set_meta/my_ds")

	// Save another version with a transform that sets the meta
	run.MustExec(t, "qri save --file testdata/movies/tf_set_meta.star test_peer_save_set_meta/my_ds")

	// Read body from the dataset that was saved.
	dsPath := run.GetPathForDataset(t, 0)
	ds := run.MustLoadDataset(t, dsPath)

	actualMetaTitle := ds.Meta.Title
	expectMetaTitle := `Did Set Title`

	// Make sure they match.
	if diff := cmp.Diff(expectMetaTitle, actualMetaTitle); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveTransformChangeMetaAndBody(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_set_meta_and_body", "qri_test_save_set_meta_and_body")
	defer run.Delete()

	// Save a first version with a normal body
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv test_peer_save_set_meta_and_body/my_ds")

	// Save another version with a transform that sets the body and a manual meta change
	err := run.ExecCommand("qri save --file testdata/movies/tf_set_len.star --file testdata/movies/meta_override.yaml test_peer_save_set_meta_and_body/my_ds")
	if err != nil {
		t.Errorf("unexpected error: %q", err)
	}
}

func TestSaveTransformConflictWithBody(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_conflict_with_body", "qri_test_save_conflict_with_body")
	defer run.Delete()

	// Save a first version with a normal body
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv test_peer_save_conflict_with_body/my_ds")

	// Save another version with a transform that sets the body and a manual body change
	err := run.ExecCommand("qri save --file testdata/movies/tf_set_len.star --body testdata/movies/body_twenty.csv test_peer_save_conflict_with_body/my_ds")
	if err == nil {
		t.Fatal("expected error trying to save, did not get an error")
	}
	expectContains := "transform script and user-supplied dataset are both trying to set:\n  body"
	if !strings.Contains(err.Error(), expectContains) {
		t.Errorf("expected error to contain %q, but got %s", expectContains, err.Error())
	}
}

func TestSaveTransformConflictWithMeta(t *testing.T) {
	run := NewTestRunner(t, "test_peer_save_conflict_with_meta", "qri_test_save_conflict_with_meta")
	defer run.Delete()

	// Save a first version with a normal body
	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv test_peer_save_conflict_with_meta/my_ds")

	// Save another version with a transform that sets the meta and a manual meta change
	err := run.ExecCommand("qri save --file testdata/movies/tf_set_meta.star --file testdata/movies/meta_override.yaml test_peer_save_conflict_with_meta/my_ds")
	if err == nil {
		t.Fatal("expected error trying to save, did not get an error")
	}
	expectContains := "transform script and user-supplied dataset are both trying to set:\n  meta"
	if !strings.Contains(err.Error(), expectContains) {
		t.Errorf("expected error to contain %q, but got %s", expectContains, err.Error())
	}
}

// TODO(dustmop): Test that if the result has a different shape than the previous version,
// the error message should be reasonable and understandable
//func TestSaveWithBadShape(t *testing.T) {
//	run := NewTestRunner(t, "qri_test_save_with_bad_shape", "qri_test_save_with_bad_shape")
//	defer run.Delete()
//
//	// Save a first version with a normal body
//	run.MustExec(t, "qri save --body testdata/movies/body_ten.csv qri_test_save_with_bad_shape/my_ds")
//	// Save with a transform that results in a different shape (non tabular)
//	run.MustExec(t, "qri save --file testdata/movies/tf_123.star qri_test_save_with_bad_shape/my_ds")
//}

// Test that saving with only a readme change will succeed
func TestSaveWithReadmeFiles(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_save_readme_files", "qri_test_save_readme_files")
	defer run.Delete()

	err := run.ExecCommand("qri save --body testdata/movies/body_ten.csv me/with_readme")
	if err != nil {
		t.Errorf("expected save to succeed, got %s", err)
	}

	err = run.ExecCommand("qri save --file testdata/movies/about_movies.md me/with_readme")
	if err != nil {
		t.Errorf("expected save to succeed, got %s", err)
	}

	err = run.ExecCommand("qri save --file testdata/movies/more_movies.md me/with_readme")
	if err != nil {
		t.Errorf("expected save to succeed, got %s", err)
	}

	err = run.ExecCommand("qri save --file testdata/movies/even_more_movies.md me/with_readme")
	if err != nil {
		t.Errorf("expected save to succeed, got %s", err)
	}
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	var qerr qrierr.Error
	if errors.As(err, &qerr) {
		return qerr.Message()
	}
	return err.Error()
}
