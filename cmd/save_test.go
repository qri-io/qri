package cmd

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/errors"
)

func TestSaveComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_save_complete")
	defer run.Delete()

	f, err := NewTestFactory()
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
		{[]string{"test", "test2"}, "test", ""},
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

		if opt.DatasetRequests == nil {
			t.Errorf("case %d, opt.DatasetRequests not set.", i)
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

		if libErr, ok := err.(errors.Error); ok {
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

	f, err := NewTestFactory()
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
		publish     bool
		dryrun      bool
		noRender    bool
		expect      string
		err         string
		msg         string
	}{
		{"no data", "me/bad_dataset", "", "", "", "", false, false, true, "", "no changes to save", ""},
		{"bad dataset file", "me/cities", "bad/filpath.json", "", "", "", false, false, true, "", "open bad/filpath.json: no such file or directory", ""},
		{"bad body file", "me/cities", "", "bad/bodypath.csv", "", "", false, false, true, "", "opening dataset.bodyPath 'bad/bodypath.csv': path not found", ""},
		{"good inputs, dryrun", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_ten.csv", "", "", false, true, true, "dataset saved: peer/movies@/map/QmWehMxKs9dFqAxjh69FKyUmXVNQRnCZ4t6quvaZ8cA8s3\nthis dataset has 1 validation errors\n", "", ""},
		{"good inputs", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_ten.csv", "", "", true, false, true, "dataset saved: peer/movies@/map/QmRgRuwLP3aZqktWv9Cv6tGwatyRjKDzqV1dDBFupJNiqj\nthis dataset has 1 validation errors\n", "", ""},
		{"add rows, dry run", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_twenty.csv", "Added 10 more rows", "Adding to the number of rows in dataset", false, true, true, "dataset saved: peer/movies@/map/QmUPXbE9rg8K7Hw71eFxYe5ky6cSaXjEraZta1YywvePBe\nthis dataset has 1 validation errors\n", "", ""},
		{"add rows, save", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_twenty.csv", "Added 10 more rows", "Adding to the number of rows in dataset", true, false, true, "dataset saved: peer/movies@/map/QmYvRp667oRMnWVGwnUD5GwceVYz2woYH9s2NkiZY2CX1f\nthis dataset has 1 validation errors\n", "", ""},
		{"no changes", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_twenty.csv", "trying to add again", "hopefully this errors", false, false, true, "", "error saving: no changes", ""},
		{"add viz", "me/movies", "testdata/movies/dataset_with_viz.json", "", "", "", false, false, false, "dataset saved: peer/movies@/map/QmVtFptuccDEKX6oY9rZsyvxTcPy1x2grwnvVfXmV8GFHA\nthis dataset has 1 validation errors\n", "", ""},
	}

	for _, c := range cases {
		run.IOReset()
		dsr, err := f.DatasetRequests()
		if err != nil {
			t.Errorf("case \"%s\", error creating dataset request: %s", c.description, err)
			continue
		}

		pathList := []string{}
		if c.filepath != "" {
			pathList = []string{c.filepath}
		}

		opt := &SaveOptions{
			IOStreams:       run.Streams,
			Refs:            NewExplicitRefSelect(c.ref),
			FilePaths:       pathList,
			BodyPath:        c.bodypath,
			Title:           c.title,
			Message:         c.message,
			Publish:         c.publish,
			DryRun:          c.dryrun,
			NoRender:        c.noRender,
			DatasetRequests: dsr,
		}

		err = opt.Run()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case '%s', mismatched error. Expected: '%s', Got: '%v'", c.description, c.err, err)
			continue
		}

		if libErr, ok := err.(errors.Error); ok {
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
			"dataset saved: test_peer/ten_movies@/ipfs/QmdYaCFsHcGiGcTG3vy2TRrDAoJ2iXyskq4gqmcfipQp9d\nthis dataset has 1 validation errors\n",
		},
		{
			"dataset file me ref",
			"qri save --file dataset.yaml me/my_dataset",
			"dataset saved: test_peer/my_dataset@/ipfs/Qmf8BV8oXo9rPmZHeo6j1BSjHFb77DEc6mNzkyXQ8wnuAK\nthis dataset has 1 validation errors\n",
		},
		{
			"dataset file explicit ref",
			"qri save --file dataset.yaml test_peer/my_dataset",
			"dataset saved: test_peer/my_dataset@/ipfs/QmeoNwgqDN133m5MZeTDt8b3W4sQQLZgDx2EkPiyxB62jY\nthis dataset has 1 validation errors\n",
		},
		{
			"body file infer name",
			"qri save --body body_ten.csv",
			"dataset saved: test_peer/body_tencsv@/ipfs/QmPA1xnbpqGzYoZNz26PcyVCgvBgWsi7hPFNvwcbamriBm\nthis dataset has 1 validation errors\n",
		},
		{
			"body file me ref",
			"qri save --body body_ten.csv me/my_dataset",
			"dataset saved: test_peer/my_dataset@/ipfs/QmTNy5ZpjdaLLqWBuuX9GhjdoxgHdGksyhV4VFQNtXTcHu\nthis dataset has 1 validation errors\n",
		},
		{
			"body file explicit ref",
			"qri save --body body_ten.csv test_peer/my_dataset",
			"dataset saved: test_peer/my_dataset@/ipfs/Qmc1JR3G3hbRdgLdPzcV7991L7LVFDH5rgaPFY9RGtoqTs\nthis dataset has 1 validation errors\n",
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
		//	"qri save --file structure.json test_peer/my_dataset",
		//	"TODO(dustmop): Should be possible to save a dataset with structure and no body",
		//},
	}
	for _, c := range goodCases {
		t.Run(c.description, func(t *testing.T) {
			// TODO(dustmop): Would be preferable to instead have a way to clear the refstore
			run := NewTestRunner(t, "test_peer", "qri_test_save_basic")
			defer run.Delete()

			err := run.ExecCommandCombinedOutErr(c.command)
			if err != nil {
				t.Errorf("error %s\n", err)
				return
			}
			actual := parseDatasetRefFromOutput(run.GetCommandOutput())
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
			"cannot save using a different username than \"test_peer\"",
		},
		{
			"dataset file explicit version",
			"qri save --file dataset.yaml me/my_dataset@/ipfs/QmVersion",
			"ref can only have username/name",
		},
		{
			"body file other username",
			"qri save --body body_ten.csv other/my_dataset",
			"cannot save using a different username than \"test_peer\"",
		},
	}
	for _, c := range badCases {
		t.Run(c.description, func(t *testing.T) {
			run := NewTestRunner(t, "test_peer", "qri_test_save_basic")
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
	run := NewTestRunner(t, "test_peer", "qri_test_save_infer_name")
	defer run.Delete()

	// Save a dataset with an inferred name.
	output := run.MustExecCombinedOutErr(t, "qri save --body testdata/movies/body_four.json")
	actual := parseDatasetRefFromOutput(output)
	expect := "dataset saved: test_peer/body_fourjson@/ipfs/QmWS1sZzkpdUiuX17b6FZEqJQGLtuxbiHg3QKtymH4dpxL\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save again, get an error because the inferred name already exists.
	err := run.ExecCommand("qri save --body testdata/movies/body_four.json")
	expectErr := `inferred dataset name already exists. To add a new commit to this dataset, run save again with the dataset reference. To create a new dataset, use --new flag`
	if err == nil {
		t.Errorf("error expected, did not get one")
	}
	if diff := cmp.Diff(expectErr, err.Error()); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save but ensure a new dataset is created.
	output = run.MustExecCombinedOutErr(t, "qri save --body testdata/movies/body_four.json --new")
	actual = parseDatasetRefFromOutput(output)
	expect = "dataset saved: test_peer/body_fourjson_1@/ipfs/QmUVzyXdUzaDa43UNXNoA4bnV1UmvmF8TQkrYUL57uRVUt\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save once again.
	output = run.MustExecCombinedOutErr(t, "qri save --body testdata/movies/body_four.json --new")
	actual = parseDatasetRefFromOutput(output)
	expect = "dataset saved: test_peer/body_fourjson_2@/ipfs/QmagSArPqsugSno5nZAz14ndeamU8CjT97rukPCYjsuTZ8\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save a dataset whose body filename starts with a number
	output = run.MustExecCombinedOutErr(t, "qri save --body testdata/movies/2018_winners.csv")
	actual = parseDatasetRefFromOutput(output)
	expect = "dataset saved: test_peer/winnerscsv@/ipfs/Qmf9yTULephFi5j8eejVLokG3c5icaY2BNJKoZiX4LRPyY\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save a dataset whose body filename is non-alphabetic
	output = run.MustExecCombinedOutErr(t, "qri save --body testdata/2015-09-16--2016-09-30.csv")
	actual = parseDatasetRefFromOutput(output)
	expect = "dataset saved: test_peer/dataset_09_16_2016_09_30csv@/ipfs/QmTbL4rDZMSC5TNoYsbwoAUihPy9RRPTWE4zGjRepUgiGZ\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveDscacheFirstCommit(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_dscache_first")
	defer run.Delete()

	// Save a dataset with one version.
	run.MustExec(t, "qri save --body testdata/movies/body_two.json me/movie_ds --use-dscache")

	// Access the dscache
	repo, err := run.RepoRoot.Repo()
	if err != nil {
		t.Fatal(err)
	}
	cache := repo.Dscache()

	// Dscache should have one reference. It has topIndex 1 because there are two logbook
	// elements in the branch, one for "init", one for "commit".
	actual := cache.VerboseString(false)
	expect := `Dscache:
 Dscache.Users:
  0) user=test_peer profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = vkys37xzcxpmw5zexzhyhpok3whl2vfeep2tyeegwnm2cxrr3umq
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = movie_ds
     bodySize      = 79
     bodyRows      = 2
     commitTime    = 978310921
     headRef       = /ipfs/QmdCjShoT9bKFjAMJ7dvdrHY476qPTouBmDs94RxMKyjms
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save a different dataset, but dscache already exists.
	run.MustExec(t, "qri save --body testdata/movies/body_four.json me/another_ds --use-dscache")

	// Because this test is using a memrepo, but the command runner instantiates its own repo
	// the dscache is not reloaded. Manually reload it here by constructing a dscache from the
	// same filename.
	fs := localfs.NewFS()
	cacheFilename := cache.Filename
	ctx := context.Background()
	// TODO(dustmop): Do we need to pass a book?
	cache = dscache.NewDscache(ctx, fs, nil, cacheFilename)

	// Dscache should have two entries now. They are alphabetized by pretty name, and have all
	// the expected data.
	actual = cache.VerboseString(false)
	expect = `Dscache:
 Dscache.Users:
  0) user=test_peer profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = zasmtvpvvddt536qmjtf4qdxszlpfcc6bqvmiemsmocaj4e74eiq
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = another_ds
     bodySize      = 137
     bodyRows      = 4
     commitTime    = 978311161
     headRef       = /ipfs/QmSHzQVuWnBpapv4LVvfYqfQhJT8VTQ4dzPaxJvPGoUrwx
  1) initID        = vkys37xzcxpmw5zexzhyhpok3whl2vfeep2tyeegwnm2cxrr3umq
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = movie_ds
     bodySize      = 79
     bodyRows      = 2
     commitTime    = 978310921
     headRef       = /ipfs/QmdCjShoT9bKFjAMJ7dvdrHY476qPTouBmDs94RxMKyjms
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveDscacheExistingDataset(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_save_dscache")
	defer run.Delete()

	// Save a dataset with one version.
	run.MustExec(t, "qri save --body testdata/movies/body_two.json me/movie_ds")

	// List with the --use-dscache flag, which builds the dscache from the logbook.
	run.MustExec(t, "qri list --use-dscache")

	// Access the dscache
	repo, err := run.RepoRoot.Repo()
	if err != nil {
		t.Fatal(err)
	}
	cache := repo.Dscache()

	// Dscache should have one reference. It has topIndex 1 because there are two logbook
	// elements in the branch, one for "init", one for "commit".
	actual := cache.VerboseString(false)
	expect := `Dscache:
 Dscache.Users:
  0) user=test_peer profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = vkys37xzcxpmw5zexzhyhpok3whl2vfeep2tyeegwnm2cxrr3umq
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = movie_ds
     bodySize      = 79
     bodyRows      = 2
     commitTime    = 978310921
     headRef       = /ipfs/QmdCjShoT9bKFjAMJ7dvdrHY476qPTouBmDs94RxMKyjms
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save a new new commit. Since the dscache exists, it should get updated.
	run.MustExec(t, "qri save --body testdata/movies/body_four.json me/movie_ds")

	// Because this test is using a memrepo, but the command runner instantiates its own repo
	// the dscache is not reloaded. Manually reload it here by constructing a dscache from the
	// same filename.
	fs := localfs.NewFS()
	cacheFilename := cache.Filename
	ctx := context.Background()
	cache = dscache.NewDscache(ctx, fs, nil, cacheFilename)

	// Dscache should now have one reference. Now topIndex is 2 because there is another "commit".
	actual = cache.VerboseString(false)
	// TODO(dlong): bodySize, bodyRows, commitTime should all be filled in
	expect = `Dscache:
 Dscache.Users:
  0) user=test_peer profileID=QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
 Dscache.Refs:
  0) initID        = vkys37xzcxpmw5zexzhyhpok3whl2vfeep2tyeegwnm2cxrr3umq
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 2
     cursorIndex   = 2
     prettyName    = movie_ds
     bodySize      = 137
     bodyRows      = 4
     commitTime    = 978311161
     headRef       = /ipfs/QmZa3VTH759etCwCgWymbgtbmoSnpRb955iWwv7uwWwCyJ
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveBadCaseCantBeUsedForNewDatasets(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_save_bad_case")
	defer run.Delete()

	// Try to save a new dataset, but its name has upper-case characters.
	err := run.ExecCommand("qri save --body testdata/movies/body_two.json test_peer/a_New_Dataset")
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
	ctx := context.Background()
	run.AddDatasetToRefstore(ctx, t, "test_peer/a_New_Dataset", &ds)

	// Save the dataset, which will work now that a version already exists.
	run.MustExec(t, "qri save --body testdata/movies/body_two.json test_peer/a_New_Dataset")
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
