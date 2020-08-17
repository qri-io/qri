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
  path: /ipfs/QmT5s5yoeYdFYAXB1Ytt9Zz6wwjcFd1Kat3h6qkLPeRQ5t
  qri: cm:0
  signature: eLr+Pk3wg8JSoeARoelNzdKLeiYFqL1k87YpXtSXigO39cAXFHg8FQki/+zt+gLPCBmPfD/mfQbR3R2mQmxyR4F+wQ1wbwkzJfumWbluyPfBsKbPQ55XLhkYUg6Ho5HqVsBn7sZ28WJ/1+GPC5xMaFYeMLTRnD2jflB5NR33eYxVQL/IDUybhgmlV5D2GOeZAOjQcCtgf0Me8o/HMqn9TsW9mNzilf7GF7lxyV+Jrz1pYjvOPlXoJGqRGRBnDbIzvEwmH7XBGHZvsbCntFw1XEEJIQslTm5mFbVrGTzqvmsPIB/SfbTjWL4Elq4uAocr7Mnu9TFKD15XNaZ8pMhaQw==
  timestamp: "2001-01-01T01:02:01.000000001Z"
  title: body changed by 54%
name: my_ds
path: /ipfs/QmRbGnuVAh83wSwa45Kc1L2erMuV5PfGxTeeft4sNeZdtx
peername: test_peer_get
previousPath: /ipfs/QmNX9ZKXtdskpYSQ5spd1qvqB2CPoWfJbdAcWoFndintrF
qri: ds:0
structure:
  checksum: QmSa4i985cF3dxNHxD5mSN7c6q1eYa83uNo1pLRmPZgTsa
  depth: 2
  entries: 18
  errCount: 1
  format: csv
  formatConfig:
    headerRow: true
    lazyQuotes: true
  length: 532
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
  path: /ipfs/QmQRZm2XT93N5T6gL3bNBmzVhjC1CU3BYthNCkBycE7R6K
  qri: cm:0
  signature: I/nrDkgwt1IPtdFKvgMQAIRYvOqKfqm6x0qfpuJ14rEtO3+uPnY3K5pVDMWJ7K+pYJz6fyguYWgXHKkbo5wZl0ICVyoIiPa9zIVbqc1d6j1v13WqtRb0bn1CXQvuI6HcBhb7+VqkSW1m+ALpxhNQuI4ZfRv8Nm8MbEpL6Ct55fJpWX1zszJ2rQP1LcH2AlEZ8bl0qpcFMk03LENUHSt1DjlaApxrEJzDgAs5drfndxXgGKYjPpkjdF+qGhn2ALV2tC64I5aIn1SJPAQnVwprUr1FmVZjZcF9m9r8WnzQ6ldj29eZIciiFlT4n2Cbw+dgPo/hNRsgzn7Our2a6r5INw==
  timestamp: "2001-01-01T01:01:01.000000001Z"
  title: created dataset from body_ten.csv
name: my_ds
path: /ipfs/QmNX9ZKXtdskpYSQ5spd1qvqB2CPoWfJbdAcWoFndintrF
peername: test_peer_get
qri: ds:0
structure:
  checksum: QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf
  depth: 2
  entries: 8
  errCount: 1
  format: csv
  formatConfig:
    headerRow: true
    lazyQuotes: true
  length: 224
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
