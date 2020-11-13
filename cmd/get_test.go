package cmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer_get", "qri_test_get_complete")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args     []string
		selector string
		refs     []string
		err      string
	}{
		{[]string{}, "", []string{}, ""},
		{[]string{"one arg"}, "", []string{"one arg"}, ""},
		{[]string{"commit", "peer/ds"}, "commit", []string{"peer/ds"}, ""},
		{[]string{"commit.author", "peer/ds"}, "commit.author", []string{"peer/ds"}, ""},
		// TODO(dlong): Fix tests when `qri get` can be passed multiple arguments.
		//{[]string{"peer/ds_two", "peer/ds"}, "", []string{"peer/ds_two", "peer/ds"}, ""},
		//{[]string{"foo", "peer/ds"}, "", []string{"foo", "peer/ds"}, ""},
		{[]string{"structure"}, "structure", []string{}, ""},
		{[]string{"stats", "me/cities"}, "stats", []string{"me/cities"}, ""},
		{[]string{"stats", "me/sitemap"}, "stats", []string{"me/sitemap"}, ""},
	}

	for i, c := range cases {
		opt := &GetOptions{
			IOStreams: run.Streams,
		}

		opt.Complete(f, c.args)

		if c.err != run.ErrStream.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, run.ErrStream.String())
			run.IOReset()
			continue
		}

		if !testSliceEqual(c.refs, opt.Refs.RefList()) {
			t.Errorf("case %d, opt.Refs not set correctly. Expected: '%q', Got: '%q'", i, c.refs, opt.Refs.RefList())
			run.IOReset()
			continue
		}

		if c.selector != opt.Selector {
			t.Errorf("case %d, opt.Selector not set correctly. Expected: '%s', Got: '%s'", i, c.selector, opt.Selector)
			run.IOReset()
			continue
		}

		if opt.DatasetMethods == nil {
			t.Errorf("case %d, opt.DatasetRequests not set.", i)
			run.IOReset()
			continue
		}
		run.IOReset()
	}
}

const (
	currHeadRepo = `bodyPath: /ipfs/QmeLmPMNSCxVxCdDmdunBCfiN1crb3C2eUnZex6QgHpFiB
commit:
  author:
    id: QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  message: "body:\n\tchanged by 54%"
  path: /ipfs/QmPid9gqi9W7pAKZT7YewqYEcnWJZW4Xs4F5a6HupWmb7S
  qri: cm:0
  signature: FALUQ3vckZ3Blf0sgxP7DZW1qIUZQn8Wnf5TQyv1F18g854Py89g+WmoJGbTX/NJEzRmsqfOvRRD6828Syhf9ubT81LnweRi6nEjWdn1JjSwvREFEjkvpxmGEyLAA6eVUfC7Oqxldl3BaBKXwmKCuw7wYpo4/6DJ+zRH1LtfVfnkbhEAzw7TiTigPT/2s4FbnWnzwMnLIkHHJF1h8HhnWLYCSWqVlyoWa42UqagsISzNvMyLSE+JCoKIuShJy1gHu7fcqFoBl5fdXbYD03Eqzmj4EY6DXMsoD8rSWzOzHC6Fpqgz6R0mkTmkDo6yXSsDo2yifrIZDJeaniI61oesaw==
  timestamp: "2001-01-01T01:02:01.000000001Z"
  title: body changed by 54%
name: my_ds
path: /ipfs/QmPyMUN7tkRhotRxyCQghLygz3gRBLo9FTFf5XwLfo55Ld
peername: test_peer_get
previousPath: /ipfs/Qmb6HLvyhZe768NLRjxDUe3zMA75yWiex7Kwq86pipQGBf
qri: ds:0
stats:
  path: /ipfs/QmXyYhG5SvTyazEd8iEjxJ7DpzKm86WwWsYyL9xMjXpjQj
  qri: sa:0
  stats:
  - count: 18
    frequencies: {}
    maxLength: 55
    minLength: 7
    type: string
  - count: 17
    histogram:
      bins: null
      frequencies: []
    max: 183
    mean: 2566
    min: 100
    type: numeric
structure:
  depth: 2
  entries: 18
  errCount: 1
  format: csv
  formatConfig:
    headerRow: true
    lazyQuotes: true
  length: 532
  path: /ipfs/QmcC8oazbAwnvZEv1k2sMXD78hZYoEoPDG1wP595Yyarg7
  qri: st:0
  schema:
    items:
      items:
      - title: movie_title
        type: string
      - title: duration
        type: integer
      type: array
    type: array

`
	prevHeadRepo = `bodyPath: /ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn
commit:
  author:
    id: QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  message: created dataset from body_ten.csv
  path: /ipfs/QmRG3g7PPKYRbusncBSk6KrQVgwr351F32VbAaBWrKDf6P
  qri: cm:0
  signature: afeBsx68Q6rN5YVeTWFqWTRE1Sbvh18m/u/kJiXz4OxvC3z4JFJJiyCNwvHH9qO8M3A48nPt5vf9JDHmO13gi0FLMz1jIWz/JTZ4aMkHLxtVKWU3gB+O7cNKFGR2FWf9SvcBiu91tBuUratshBHlKwB2GGim1MhAHIc3kuVsX21ytLOFDYelHK8FCiurBcOm3/HwWhOFuJ9OKFK47jnTDwub7x1FA2GhQ3fuYdfLD4DeAjj21V7sVA2Du6ady1RDYHE2P3wwVPG9ysWIVq59DvBPGNuHbbktaErBHcM9rhq8Uzeoj2Uo8d0sSa1NVKvlRcFan4MlYv5BlmzXjrdK2A==
  timestamp: "2001-01-01T01:01:01.000000001Z"
  title: created dataset from body_ten.csv
name: my_ds
path: /ipfs/Qmb6HLvyhZe768NLRjxDUe3zMA75yWiex7Kwq86pipQGBf
peername: test_peer_get
qri: ds:0
stats:
  path: /ipfs/QmXHqvGRzaDUTbiKzBezR43mchpwoSE3xuf8HnfVkdfkKk
  qri: sa:0
  stats:
  - count: 8
    frequencies: {}
    maxLength: 55
    minLength: 7
    type: string
  - count: 7
    histogram:
      bins: null
      frequencies: []
    max: 178
    mean: 1047
    min: 100
    type: numeric
structure:
  depth: 2
  entries: 8
  errCount: 1
  format: csv
  formatConfig:
    headerRow: true
    lazyQuotes: true
  length: 224
  path: /ipfs/QmWqobG1jjqWnNadHYVhv9m3TXudrabLwmoybQLgvFvQa3
  qri: st:0
  schema:
    items:
      items:
      - title: movie_title
        type: string
      - title: duration
        type: integer
      type: array
    type: array

`
	currBodyRepo = `movie_title,duration
Avatar ,178
Pirates of the Caribbean: At World's End ,169
Spectre ,148
The Dark Knight Rises ,164
Star Wars: Episode VII - The Force Awakens             ,
John Carter ,132
Spider-Man 3 ,156
Tangled ,100
Avengers: Age of Ultron ,141
Harry Potter and the Half-Blood Prince ,153
Batman v Superman: Dawn of Justice ,183
Superman Returns ,169
Quantum of Solace ,106
Pirates of the Caribbean: Dead Man's Chest ,151
The Lone Ranger ,150
Man of Steel ,143
The Chronicles of Narnia: Prince Caspian ,150
The Avengers ,173

`
	prevBodyRepo = `movie_title,duration
Avatar ,178
Pirates of the Caribbean: At World's End ,169
Spectre ,148
The Dark Knight Rises ,164
Star Wars: Episode VII - The Force Awakens             ,
John Carter ,132
Spider-Man 3 ,156
Tangled ,100

`
	currHeadFSI = `bodyPath: /tmp/my_ds/my_ds/body.csv
name: my_ds
peername: test_peer_get
qri: ds:0
structure:
  format: csv
  formatConfig:
    headerRow: true
    lazyQuotes: true
  qri: st:0
  schema:
    items:
      items:
      - title: movie_title
        type: string
      - title: duration
        type: integer
      type: array
    type: array

`
	currBodyFSI = currBodyRepo
)

func TestGetDatasetFromRepo(t *testing.T) {
	run := NewTestRunner(t, "test_peer_get", "get_dataset_head")
	defer run.Delete()

	// Save two versions.
	got := run.MustExecCombinedOutErr(t, "qri save --body=testdata/movies/body_ten.csv me/my_ds")
	ref := parseRefFromSave(got)
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/my_ds")

	// Get head.
	output := run.MustExec(t, "qri get me/my_ds")
	expect := currHeadRepo
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get one version ago.
	output = run.MustExec(t, fmt.Sprintf("qri get %s", ref))
	expect = prevHeadRepo
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get body from current commit.
	output = run.MustExec(t, "qri get body me/my_ds")
	expect = currBodyRepo
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get body from one version ago.
	output = run.MustExec(t, fmt.Sprintf("qri get body %s", ref))
	expect = prevBodyRepo
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}

func TestGetDatasetCheckedOut(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_get", "get_dataset_checked_out")
	defer run.Delete()

	// Save two versions.
	got := run.MustExecCombinedOutErr(t, "qri save --body=testdata/movies/body_ten.csv me/my_ds")
	ref := parseRefFromSave(got)
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/my_ds")

	// Checkout to a working directory.
	run.CreateAndChdirToWorkDir("my_ds")
	run.MustExec(t, "qri checkout me/my_ds")

	// Get head.
	output := run.MustExec(t, "qri get me/my_ds")
	expect := currHeadFSI
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get one version ago.
	output = run.MustExec(t, fmt.Sprintf("qri get %s", ref))
	expect = prevHeadRepo
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get body from current commit.
	output = run.MustExec(t, "qri get body me/my_ds")
	expect = currBodyFSI
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get body from one version ago.
	output = run.MustExec(t, fmt.Sprintf("qri get body %s", ref))
	expect = prevBodyRepo
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}

func TestGetDatasetUsingDscache(t *testing.T) {
	run := NewTestRunner(t, "test_peer_get", "get_dataset_head")
	defer run.Delete()

	// Save two versions, using dscache.
	got := run.MustExecCombinedOutErr(t, "qri save --use-dscache --body=testdata/movies/body_ten.csv me/my_ds")
	ref := parseRefFromSave(got)
	run.MustExec(t, "qri save --use-dscache --body=testdata/movies/body_twenty.csv me/my_ds")

	// Get head.
	output := run.MustExec(t, "qri get me/my_ds")
	expect := currHeadRepo
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get one version ago.
	output = run.MustExec(t, fmt.Sprintf("qri get %s", ref))
	expect = prevHeadRepo
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get body from current commit.
	output = run.MustExec(t, "qri get body me/my_ds")
	expect = currBodyRepo
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get body from one version ago.
	output = run.MustExec(t, fmt.Sprintf("qri get body %s", ref))
	expect = prevBodyRepo
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}

func TestGetDatasetCheckedOutUsingDscache(t *testing.T) {
	run := NewFSITestRunner(t, "test_peer_get", "get_dataset_checked_out_using_dscache")
	defer run.Delete()

	// Save two versions.
	got := run.MustExecCombinedOutErr(t, "qri save --body=testdata/movies/body_ten.csv me/my_ds")
	ref := parseRefFromSave(got)
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/my_ds")

	// Checkout to a working directory.
	run.CreateAndChdirToWorkDir("my_ds")
	run.MustExec(t, "qri checkout me/my_ds")

	// Build the dscache
	// TODO(dustmop): Can't immitate the other tests, because checkout doesn't know about dscache
	// yet, it doesn't set the FSIPath. Instead, build the dscache here, so that the FSIPath exists.
	run.MustExec(t, "qri list --use-dscache")

	// Get head.
	output := run.MustExec(t, "qri get me/my_ds")
	expect := currHeadFSI
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get one version ago.
	output = run.MustExec(t, fmt.Sprintf("qri get %s", ref))
	expect = prevHeadRepo
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get body from current commit.
	output = run.MustExec(t, "qri get body me/my_ds")
	expect = currBodyFSI
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get body from one version ago.
	output = run.MustExec(t, fmt.Sprintf("qri get body %s", ref))
	expect = prevBodyRepo
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}

func TestGetRemoteDataset(t *testing.T) {
	run := NewTestRunnerWithMockRemoteClient(t, "test_get_remote_dataset", "get_remote_dataset")
	defer run.Delete()

	expect := "cannot use '--offline' and '--remote' flags together"
	err := run.ExecCommand("qri get --remote=registry --offline other_peer/their_dataset")
	if expect != err.Error() {
		t.Errorf("response mismatch\nwant: %q\n got: %q", expect, err)
	}

	expect = "reference not found"
	err = run.ExecCommand("qri get --offline other_peer/their_dataset")
	if expect != err.Error() {
		t.Errorf("response mismatch\nwant: %q\n got: %q", expect, err)
	}

	expect = "bodyPath: /ipfs/QmbJWAESqCsf4RFCqEY7jecCashj8usXiyDNfKtZCwwzGb\ncommit:\n  message: created dataset\n  path: /ipfs/QmRXxnXPRzorHuWw5CRa9YJFHXz85gjbrASvh24vgUubRg\n  qri: cm:0\n  signature: QGFDik4Izple4Rm3giNpc0/sNyxA4ryOdlP8VIZ0GiNRpx5h+F7smLYFczRxV8tunzqZsq1jS+r5xHo3wBk6k+oAMm6C/LGyr96c0o9R2iEpFUTmHBr9zP2Dicyc4T4cZmz/Kyg/wT0MZOMHPnzujfElb6PfR3INHtagZYEd/kA3NaXm6RqmQnz+SCJaphTyrgmzH5vxtl2Gd377HCcLL/CvywhrnRQ9h0tsr6pkbxXyelagqDnBiiK+INAo089i3XKBTu7dnwAlVxzGxY3vCK7YRZQhq5qwJiBCU0tos0HDljya/lFc9rNMLzeuJUzpmABMJ7PrTm7ZglZRb03Pmg==\n  timestamp: \"2001-01-01T01:01:01.000000001Z\"\n  title: created dataset\nname: their_dataset\npath: /ipfs/Qmb6NpFCzEBtPGZdp8QasFGLM6AxPEBRStB4jL4BWW9oBG\npeername: other_peer\nqri: ds:0\nstats: /ipfs/QmQQkQF2KNBZfFiX33jJ9hu6ivfoHrtgcwMRAezS4dcA7c\nstructure:\n  depth: 1\n  format: json\n  length: 2\n  path: /ipfs/QmUpFYQZjRjpN8gweCZeFPrCHnUbqwJBnXajnsbymPY45c\n  qri: st:0\n  schema:\n    type: object\n\n"
	got := run.MustExec(t, "qri get --remote=registry other_peer/their_dataset")
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("repsonse mismatch (-want +got):\n%s", diff)
	}
}
