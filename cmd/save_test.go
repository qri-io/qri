package cmd

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/startf"
)

func TestSaveComplete(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

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
			IOStreams: streams,
		}

		opt.Complete(f, c.args)

		if c.err != errs.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
			ioReset(in, out, errs)
			continue
		}

		if c.expect != opt.Refs.Ref() {
			t.Errorf("case %d, opt.Ref not set correctly. Expected: '%s', Got: '%s'", i, c.expect, opt.Refs.Ref())
			ioReset(in, out, errs)
			continue
		}

		if opt.DatasetRequests == nil {
			t.Errorf("case %d, opt.DatasetRequests not set.", i)
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
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

		if libErr, ok := err.(lib.Error); ok {
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
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)
	setNoPrompt(true)

	prevXformVer := startf.Version
	startf.Version = "test_version"
	defer func() {
		startf.Version = prevXformVer
	}()

	// to keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

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
		{"good inputs, dryrun", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_ten.csv", "", "", false, true, true, "dataset saved: peer/movies@/map/QmYPf7XVDXPB3hQy8ptetyXAhGJsFdJmKfBrN9vWWvG3eQ\nthis dataset has 1 validation errors\n", "", ""},
		{"good inputs", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_ten.csv", "", "", true, false, true, "dataset saved: peer/movies@/map/QmYPf7XVDXPB3hQy8ptetyXAhGJsFdJmKfBrN9vWWvG3eQ\nthis dataset has 1 validation errors\n", "", ""},
		{"add rows, dry run", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_twenty.csv", "Added 10 more rows", "Adding to the number of rows in dataset", false, true, true, "dataset saved: peer/movies@/map/Qmf516nREprnwE3YnBpZd1y9DkS6e2ktKYDLMtz4L4MgMX\nthis dataset has 1 validation errors\n", "", ""},
		{"add rows, save", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_twenty.csv", "Added 10 more rows", "Adding to the number of rows in dataset", true, false, true, "dataset saved: peer/movies@/map/Qmf516nREprnwE3YnBpZd1y9DkS6e2ktKYDLMtz4L4MgMX\nthis dataset has 1 validation errors\n", "", ""},
		{"no changes", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_twenty.csv", "trying to add again", "hopefully this errors", false, false, true, "", "error saving: no changes", ""},
		{"add viz", "me/movies", "testdata/movies/dataset_with_viz.json", "", "", "", false, false, false, "dataset saved: peer/movies@/map/QmXDKr4ryBuzXFhpKPUAe6GmqEacCjgPkG76NyVLpmPDSt\nthis dataset has 1 validation errors\n", "", ""},
	}

	for _, c := range cases {
		ioReset(in, out, errs)
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
			IOStreams:       streams,
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

		if libErr, ok := err.(lib.Error); ok {
			if libErr.Message() != c.msg {
				t.Errorf("case '%s', mismatched user-friendly message. Expected: '%s', Got: '%s'", c.description, c.msg, libErr.Message())
				continue
			}
		} else if c.msg != "" {
			t.Errorf("case '%s', mismatched user-friendly message. Expected: '%s', Got: ''", c.description, c.msg)
			continue
		}

		if c.expect != errs.String() {
			t.Errorf("case '%s', err output mismatch. Expected: '%s', Got: '%s'", c.description, c.expect, errs.String())

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
			"dataset saved: test_peer/ten_movies@/ipfs/QmU935SkFafE786u7jQPknB858PDrJRFUJboxbS4vKkAe7\nthis dataset has 1 validation errors\n",
		},
		{
			"dataset file me ref",
			"qri save --file dataset.yaml me/my_dataset",
			"dataset saved: test_peer/my_dataset@/ipfs/QmSSTAcuehjFFiHNBFicTXJv2uhbKg4hjGgCL8W46D3jGG\nthis dataset has 1 validation errors\n",
		},
		{
			"dataset file explicit ref",
			"qri save --file dataset.yaml test_peer/my_dataset",
			"dataset saved: test_peer/my_dataset@/ipfs/QmeW1M8W9yj69pxE3uJZbYVBCaEARYkrkje3NHY1s6tHtP\nthis dataset has 1 validation errors\n",
		},
		{
			"body file infer name",
			"qri save --body body_ten.csv",
			"dataset saved: test_peer/body_tencsv@/ipfs/QmWaS8ndeAhAWWSUmwJ6HiXKMNWnsL1LYhzrWJyzCgR9pd\nthis dataset has 1 validation errors\n",
		},
		{
			"body file me ref",
			"qri save --body body_ten.csv me/my_dataset",
			"dataset saved: test_peer/my_dataset@/ipfs/QmeyznW4FBaLY8rk5xpZyFHXixrq8qFJrFf8v1REZzmLLg\nthis dataset has 1 validation errors\n",
		},
		{
			"body file explicit ref",
			"qri save --body body_ten.csv test_peer/my_dataset",
			"dataset saved: test_peer/my_dataset@/ipfs/QmP9yay2KwPnyDLLhGXZsLdKG31qMAR3CSBAmscz3znC9C\nthis dataset has 1 validation errors\n",
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

			err := run.ExecCommand(c.command)
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
	output := run.MustExec(t, "qri save --body testdata/movies/body_four.json")
	actual := parseDatasetRefFromOutput(output)
	expect := "dataset saved: test_peer/body_fourjson@/ipfs/QmcHttiUsLCcjEdhuSyA1YXFPymx3k7srjUjrsJdRCPSKn\n"
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
	output = run.MustExec(t, "qri save --body testdata/movies/body_four.json --new")
	actual = parseDatasetRefFromOutput(output)
	expect = "dataset saved: test_peer/body_fourjson_1@/ipfs/QmXzwvaxmHUbjkAmirgKEzQyQAvaDRXEcNizodaJ5LqbjE\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Save once again.
	output = run.MustExec(t, "qri save --body testdata/movies/body_four.json --new")
	actual = parseDatasetRefFromOutput(output)
	expect = "dataset saved: test_peer/body_fourjson_2@/ipfs/QmbURFB5RLWN3LdtKe9yX3emiyvNtvtocsoaK8eFFVTXFL\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveDscacheFirstCommit(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_dscache_first")
	defer run.Delete()

	prevTimestampFunc := logbook.NewTimestamp
	logbook.NewTimestamp = func() int64 {
		return 1000
	}
	defer func() {
		logbook.NewTimestamp = prevTimestampFunc
	}()

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
  0) initID        = gdulisqthishkm3rrrk4sals4hnnljkgurxteqwnlq5kssxqpp3q
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = movie_ds
     bodySize      = 79
     bodyRows      = 2
     commitTime    = 978310861
     headRef       = /ipfs/QmYFApC68tU71We4rJ3Rp4k2tJhFuknfS8MvcJJfaLPAEi
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
  0) initID        = mji5zbbkphzl7unnngzzgel3yqnlzoa2x5w4segq2x2o3pv6o6da
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = another_ds
     bodySize      = 137
     bodyRows      = 4
     commitTime    = 978310921
     headRef       = /ipfs/QmVjFS46hBiyuCjg7zYUijmCCZz1GG53ZSgNFKw4hbjWnY
  1) initID        = gdulisqthishkm3rrrk4sals4hnnljkgurxteqwnlq5kssxqpp3q
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = movie_ds
     bodySize      = 79
     bodyRows      = 2
     commitTime    = 978310861
     headRef       = /ipfs/QmYFApC68tU71We4rJ3Rp4k2tJhFuknfS8MvcJJfaLPAEi
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveDscacheExistingDataset(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_save_dscache")
	defer run.Delete()

	prevTimestampFunc := logbook.NewTimestamp
	logbook.NewTimestamp = func() int64 {
		return 1000
	}
	defer func() {
		logbook.NewTimestamp = prevTimestampFunc
	}()

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
  0) initID        = gdulisqthishkm3rrrk4sals4hnnljkgurxteqwnlq5kssxqpp3q
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 1
     cursorIndex   = 1
     prettyName    = movie_ds
     bodySize      = 79
     bodyRows      = 2
     commitTime    = 978310861
     headRef       = /ipfs/QmYFApC68tU71We4rJ3Rp4k2tJhFuknfS8MvcJJfaLPAEi
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
  0) initID        = gdulisqthishkm3rrrk4sals4hnnljkgurxteqwnlq5kssxqpp3q
     profileID     = QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
     topIndex      = 2
     cursorIndex   = 2
     prettyName    = movie_ds
     bodySize      = 137
     bodyRows      = 4
     commitTime    = 978310921
     headRef       = /ipfs/QmbKb9dHzcFsngjDUUUZ12XsREuh37Zm7yJ4f3t5BkCdQv
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func TestSaveBadCaseCantBeUsedForNewDatasets(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_save_bad_case")
	defer run.Delete()

	prevTimestampFunc := logbook.NewTimestamp
	logbook.NewTimestamp = func() int64 {
		return 1000
	}
	defer func() {
		logbook.NewTimestamp = prevTimestampFunc
	}()

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
