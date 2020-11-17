package cmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset/dstest"
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
	currHeadRepo = `bodyPath: {{ .bodyPath }}
commit:
  author:
    id: {{ .profileID }}
  message: "body:\n\tchanged by 54%"
  path: {{ .commitPath }}
  qri: cm:0
  signature: {{ .signature }}
  timestamp: "2001-01-01T01:02:01.000000001Z"
  title: body changed by 54%
name: my_ds
path: {{ .path }}
peername: test_peer_get
previousPath: {{ .previousPath }}
qri: ds:0
stats:
  path: {{ .statsPath }}
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
  checksum: {{ .bodyPath }}
  depth: 2
  entries: 18
  errCount: 1
  format: csv
  formatConfig:
    headerRow: true
    lazyQuotes: true
  length: 532
  path: {{ .structurePath }}
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

	prevHeadRepo = `bodyPath: {{ .bodyPath }}
commit:
  author:
    id: {{ .profileID }}
  message: created dataset from body_ten.csv
  path: {{ .commitPath }}
  qri: cm:0
  signature: {{ .signature }}
  timestamp: "2001-01-01T01:01:01.000000001Z"
  title: created dataset from body_ten.csv
name: my_ds
path: {{ .path }}
peername: test_peer_get
qri: ds:0
stats:
  path: {{ .statsPath }}
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
  checksum: {{ .bodyPath }}
  depth: 2
  entries: 8
  errCount: 1
  format: csv
  formatConfig:
    headerRow: true
    lazyQuotes: true
  length: 224
  path: {{ .structurePath }}
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

var (
	currHeadRepoData = map[string]string{
		"profileID":     "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
		"bodyPath":      "/ipfs/QmeLmPMNSCxVxCdDmdunBCfiN1crb3C2eUnZex6QgHpFiB",
		"commitPath":    "/ipfs/Qmejjpn1mrhQVxZqppoumJ9C2ED5pyxSZFvcqrHfVVcJtU",
		"signature":     "iHj80RtzSKXII2gLS4eyheTzu10jo/C/b6RJgzqJ9fuhXMz4NrZ+vN1KM6aaC1Xogv57lPd6w1pG3oa7K2IcSHpmcbQedT4FN2TBgYzGE3w7JkXqSKG19q1rNQdFFRnBzdUWcPlVoLgTabwe79A7DUEPGTmMrtrRrjig/eXlAe+90uZNHp+SORpqRgu6GzqFWs+sy4NXBpSiIE2KsP8XIW9V/HdTEWkgDAaFJBZN4TfDkYFvimjadIYOx98miu7Jvo6dmcsdgxgykTOMiRiauL5fEb+XTgkLaUj60/i7gRmdvEo5Vhp8DPOVYBldqKQDF20cdDQScBo/WA/RYY41qg==",
		"path":          "/ipfs/QmdnKwHbQWD9QsfSyWQsZebBZQKKa5uKrxA5TAXHM1tvxL",
		"previousPath":  "/ipfs/QmcFgNLik6zDXBz2ever6o5qYdkDJ2R7ryzMHPN9esvoHK",
		"statsPath":     "/ipfs/QmXyYhG5SvTyazEd8iEjxJ7DpzKm86WwWsYyL9xMjXpjQj",
		"structurePath": "/ipfs/QmcAfMfZ7qTNiCfQxnRJyDxEDM7tqDstvpgviT73PFbabZ",
	}
	prevHeadRepoData = map[string]string{
		"profileID":     "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
		"bodyPath":      "/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn",
		"commitPath":    "/ipfs/QmYWpHLZYcnjUr2NA9BVypgb7wJnQJAriKT6EV32AG69dA",
		"signature":     "Nid1+5YRJQENq3CJDD7CXqG2bekV6hEZkE/hI93TZs4Q1xvoCG8FLGO4s6VB7yKEHnBsbLhTl654OfzF6AF1qWvV2mzgbSgRwxSIUEuP4SwvlH/9K6AvqIZwaihcAeACsXd5bgzzBz8P3CIkC9idGEiLr+xXn5EBw8VFoLtGTW+p/Ie78CYzGY3PnW5llG79qahWZJOZQJfqIk/TDfpgJFcVHfAj0GmefshTtwPfV5IHwX2dQFAuGMxyWGdcvuxZHC4kbcbnQ5Htymsr4eKVI/FRvp+T+HnRwhtoc3mo8nWi01Q5p+S9ABxHqcCSoyNWJd2VlfZknOTvcgc/SFiQOw==",
		"path":          "/ipfs/QmcFgNLik6zDXBz2ever6o5qYdkDJ2R7ryzMHPN9esvoHK",
		"previousPath":  "/ipfs/QmcFgNLik6zDXBz2ever6o5qYdkDJ2R7ryzMHPN9esvoHK",
		"statsPath":     "/ipfs/QmXHqvGRzaDUTbiKzBezR43mchpwoSE3xuf8HnfVkdfkKk",
		"structurePath": "/ipfs/QmSxuAVwd9pPf9c7WMu1gjUsHSLBLRuxQcFjyu9mfsA2TQ",
	}
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
	expect := dstest.Template(t, currHeadRepo, currHeadRepoData)
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get one version ago.
	output = run.MustExec(t, fmt.Sprintf("qri get %s", ref))
	expect = dstest.Template(t, prevHeadRepo, prevHeadRepoData)
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
	expect = dstest.Template(t, prevHeadRepo, prevHeadRepoData)
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
	expect := dstest.Template(t, currHeadRepo, currHeadRepoData)
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Get one version ago.
	output = run.MustExec(t, fmt.Sprintf("qri get %s", ref))
	expect = dstest.Template(t, prevHeadRepo, prevHeadRepoData)
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
	expect = dstest.Template(t, prevHeadRepo, prevHeadRepoData)
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

	expect = dstest.Template(t, `bodyPath: {{ .bodyPath }}
commit:
  message: created dataset
  path: {{ .commitPath }}
  qri: cm:0
  signature: {{ .signature }}
  timestamp: "2001-01-01T01:01:01.000000001Z"
  title: created dataset
name: their_dataset
path: {{ .path }}
peername: other_peer
qri: ds:0
stats: {{ .statsPath }}
structure:
  checksum: {{ .bodyPath }}
  depth: 1
  format: json
  length: 2
  path: {{ .structurePath }}
  qri: st:0
  schema:
    type: object

`, map[string]string{
		"bodyPath":      "/ipfs/QmbJWAESqCsf4RFCqEY7jecCashj8usXiyDNfKtZCwwzGb",
		"commitPath":    "/ipfs/QmTTPd47BD4EGpCpuvRwTRqDRF84iAuJmfUUGcfEBuF7he",
		"signature":     "gySMr/FiT+kz0X2ODXCE5APx/BvPvalw4xlbS8TtSWssEoHlAOdrUNKUfU7j6rjyq7sFJ7hrbIVOn87fx+7arYCvrvikRawd2anzIvIruxfBymS6A0HtAGAOEAvpn3XbDykEjqaomTXS1CyR6wQkwNEgbELCIqwda9UV3ulhUtHMrAyMxvnq3NG6J9wyFB13u133aDVEojJ82mEF5DBFB+VBVbw90S4b/5AxLEUFSt/BCtE1O0lKYCt2x0HK+1fhl85oe3fpqLhLk96qCAR/Ngv4bt0E9NjGi2ltuji8gaDICKe5KRaSXjXlMkwbUq6sXEKgqzfxHXoIAUZnZNwnmg==",
		"path":          "/ipfs/Qme666Kphnyw8Sf9sjJaEUp1gQ9PodDyZVW878b6pHny9n",
		"statsPath":     "/ipfs/QmQQkQF2KNBZfFiX33jJ9hu6ivfoHrtgcwMRAezS4dcA7c",
		"structurePath": "/ipfs/QmWoYVZWDdiNauzeP171hKSdo3p2bFaqDcW6cppb9QugUE",
	})
	got := run.MustExec(t, "qri get --remote=registry other_peer/their_dataset")
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("repsonse mismatch (-want +got):\n%s", diff)
	}
}
