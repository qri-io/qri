package cmd

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/registry/regclient"
)

func TestSearchComplete(t *testing.T) {
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
		opt := &SearchOptions{
			IOStreams: streams,
		}

		opt.Complete(f, c.args)

		if c.err != errs.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
			ioReset(in, out, errs)
			continue
		}

		if c.expect != opt.Query {
			t.Errorf("case %d, opt.Ref not set correctly. Expected: '%s', Got: '%s'", i, c.expect, opt.Query)
			ioReset(in, out, errs)
			continue
		}

		if opt.SearchMethods == nil {
			t.Errorf("case %d, opt.SearchMethods not set.", i)
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
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

// SearchTestRunner holds state used by the search test
type SearchTestRunner struct {
	Pwd      string
	RootPath string
	Teardown func()
}

// NewSearchTestRunner sets up state needed for the search test
func NewSearchTestRunner(t *testing.T) *SearchTestRunner {
	run := SearchTestRunner{}

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

func TestSearchRun(t *testing.T) {
	run := NewSearchTestRunner(t)
	defer run.Close()

	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	// mock registry server that returns cached response data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(mockResponse)
	}))
	rc := regclient.NewClient(&regclient.Config{Location: server.URL})

	f, err := NewTestFactoryInstanceOptions(lib.OptRegistryClient(rc))
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
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
		sr, err := f.SearchMethods()
		if err != nil {
			t.Errorf("case %d, error creating dataset request: %s", i, err)
			continue
		}

		opt := &SearchOptions{
			IOStreams:     streams,
			Query:         c.query,
			Format:        c.format,
			SearchMethods: sr,
		}

		err = opt.Run()

		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%v'", i, c.err, err)
			ioReset(in, out, errs)
			continue
		}

		if libErr, ok := err.(lib.Error); ok {
			if libErr.Message() != c.msg {
				t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: '%s'", i, c.msg, libErr.Message())
				ioReset(in, out, errs)
				continue
			}
		} else if c.msg != "" {
			t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: ''", i, c.msg)
			ioReset(in, out, errs)
			continue
		}

		if c.expected != out.String() {
			t.Errorf("case %d, output mismatch. Expected: '%s', Got: '%s'", i, c.expected, out.String())
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
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
    https://qri.cloud/nuun/nuun
    /ipfs/QmZEnjt3Y5RxXsoZyufJfFzcogicBEwfaimJSyDuC7nySA
    this is a d
    36 B, 3 entries, 0 errors

`

var jsonSearchResponse = `[
  {
    "Type": "dataset",
    "ID": "/ipfs/QmZEnjt3Y5RxXsoZyufJfFzcogicBEwfaimJSyDuC7nySA",
    "URL": "https://qri.cloud/nuun/nuun",
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
