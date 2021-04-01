package cmd

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/registry/regclient"
)

func TestSearchComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer_search_complete", "qri_test_search_complete")
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
		{[]string{"test", "test2"}, "test", ""},
	}

	for i, c := range cases {
		opt := &SearchOptions{
			IOStreams: run.Streams,
		}

		opt.Complete(f, c.args)

		if c.err != run.ErrStream.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, run.ErrStream.String())
			run.IOReset()
			continue
		}

		if c.expect != opt.Query {
			t.Errorf("case %d, opt.Ref not set correctly. Expected: '%s', Got: '%s'", i, c.expect, opt.Query)
			run.IOReset()
			continue
		}

		run.IOReset()
	}
}

func TestSearchValidate(t *testing.T) {
	cases := []struct {
		query string
		err   string
		msg   string
	}{
		{"test", "", ""},
		{"", lib.ErrBadArgs.Error(), "please provide search parameters, for example:\n    $ qri search census\n    $ qri search 'census 2018'\nsee `qri search --help` for more information"},
	}
	for i, c := range cases {
		opt := &SearchOptions{
			Query: c.query,
		}

		err := opt.Validate()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: %s, Got: %s", i, c.err, err)
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

// SearchTestRunner holds state used by the search test
// TODO(dustmop): Compose this with TestRunner instead
type SearchTestRunner struct {
	Pwd       string
	RootPath  string
	Teardown  func()
	Streams   ioes.IOStreams
	InStream  *bytes.Buffer
	OutStream *bytes.Buffer
	ErrStream *bytes.Buffer
}

// NewSearchTestRunner sets up state needed for the search test
// TODO (b5) - add an explicit RepoPath to the SearchTestRunner. Tests are
// relying on the "RootPath" property, which should be configurable per-test
func NewSearchTestRunner(t *testing.T) *SearchTestRunner {
	run := SearchTestRunner{}

	// Set IOStreams
	run.Streams, run.InStream, run.OutStream, run.ErrStream = ioes.NewTestIOStreams()

	// Get current directory
	var err error
	run.Pwd, err = os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Create temporary directory to run the test in
	run.RootPath, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	os.Chdir(run.RootPath)

	// Clean up function
	run.Teardown = func() {
		os.Chdir(run.Pwd)
		os.RemoveAll(run.RootPath)
	}
	return &run
}

// Close tears down the test
func (r *SearchTestRunner) Close() {
	r.Teardown()
}

// IOReset resets the io streams
func (r *SearchTestRunner) IOReset() {
	r.InStream.Reset()
	r.OutStream.Reset()
	r.ErrStream.Reset()
}

func TestSearchRun(t *testing.T) {
	run := NewSearchTestRunner(t)
	defer run.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setNoColor(true)

	// mock registry server that returns cached response data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(mockResponse)
	}))
	rc := regclient.NewClient(&regclient.Config{Location: server.URL})

	f, err := NewTestFactoryInstanceOptions(ctx, run.RootPath, lib.OptRegistryClient(rc))
	if err != nil {
		t.Fatal(err)
	}
	inst, err := f.Instance()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		query    string
		format   string
		expected string
		err      string
		msg      string
	}{
		{"test", "", textSearchResponse, "", ""},
		{"test", "json", jsonSearchResponse, "", ""},
	}

	for i, c := range cases {
		opt := &SearchOptions{
			IOStreams: run.Streams,
			Query:     c.query,
			Format:    c.format,
			Instance:  inst,
		}

		err = opt.Run()

		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%v'", i, c.err, err)
			run.IOReset()
			continue
		}

		if libErr, ok := err.(errors.Error); ok {
			if libErr.Message() != c.msg {
				t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: '%s'", i, c.msg, libErr.Message())
				run.IOReset()
				continue
			}
		} else if c.msg != "" {
			t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: ''", i, c.msg)
			run.IOReset()
			continue
		}

		if c.expected != run.OutStream.String() {
			t.Errorf("case %d, output mismatch. Expected: '%s', Got: '%s'", i, c.expected, run.OutStream.String())
			run.IOReset()
			continue
		}
		run.IOReset()
	}
}

var mockResponse = []byte(`{"data":[
	{
		"commit": {
			"qri": "cm:0",
			"timestamp": "2019-08-31T12:07:56.212858Z",
			"title": "change to 10"
		},
		"meta": {
			"keywords": [
				"joke"
			],
			"qri": "md:0",
			"title": "this is a d"
		},
		"name": "nuun",
		"path": "/ipfs/QmZEnjt3Y5RxXsoZyufJfFzcogicBEwfaimJSyDuC7nySA",
		"peername": "nuun",
		"qri": "ds:0",
		"structure": {
			"entries": 3,
			"format": "csv",
			"length": 36,
			"qri": "st:0"
		}
	}
],"meta":{"code":200}}`)

var textSearchResponse = `showing 1 results for 'test'
1   nuun/nuun
    /ipfs/QmZEnjt3Y5RxXsoZyufJfFzcogicBEwfaimJSyDuC7nySA
    this is a d
    36 B, 3 entries, 0 errors

`

var jsonSearchResponse = `[
  {
    "Type": "dataset",
    "ID": "/ipfs/QmZEnjt3Y5RxXsoZyufJfFzcogicBEwfaimJSyDuC7nySA",
    "URL": "",
    "Value": {
      "commit": {
        "qri": "cm:0",
        "timestamp": "2019-08-31T12:07:56.212858Z",
        "title": "change to 10"
      },
      "meta": {
        "keywords": [
          "joke"
        ],
        "qri": "md:0",
        "title": "this is a d"
      },
      "name": "nuun",
      "path": "/ipfs/QmZEnjt3Y5RxXsoZyufJfFzcogicBEwfaimJSyDuC7nySA",
      "peername": "nuun",
      "qri": "ds:0",
      "structure": {
        "entries": 3,
        "format": "csv",
        "length": 36,
        "qri": "st:0"
      }
    }
  }
]`
