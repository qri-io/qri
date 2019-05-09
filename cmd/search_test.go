package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/registry/regclient"
)

func TestSearchComplete(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory(nil)
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

		if opt.SearchRequests == nil {
			t.Errorf("case %d, opt.SearchRequests not set.", i)
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

func TestSearchRun(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	// mock registry server that returns fake response data
	var mockResponse = []byte(`{"data":[{"Type":"","ID":"ramfox/test","Value":null},{"Type":"","ID":"b5/test","Value":{"commit":{"path":"/","signature":"JI/VSNqMuFGYVEwm3n8ZMjZmey+W2mhkD5if2337wDp+kaYfek9DntOyZiILXocW5JuOp48EqcsWf/BwhejQiYZ2utaIzR8VcMPo7u7c5nz2G6JTsoW+u9UUaKRVtl30jh6Kg1O2bnhYh9v4qW9VQgxOYfdhBl6zT4cYcjm1UkrblEe/wh494k9NziM5Bi2ATGRE2m71Lsf/TEDoNI549SebLQ1dsWXr1kM7lCeFqlDVjgbKQmGXowqcK/P9v+RBIRCnArnwFe/BQq4i1wmmnMEqpuUnfWR3xfJTE1DUMVaAid7U0jTWGVxROUdKk6mmTzlb1PiNdfruP+SFhjyQwQ==","timestamp":"2018-05-25T13:44:54.692493401Z","title":"created dataset"},"meta":{"citations":[{"url":"https://api.github.com/repos/qri-io/qri/releases"}],"qri":"md:0"},"path":"/ipfs/QmPi5wrPsY4xPwy2oRr7NRZyfFxTeupfmnrVDubzoABLNP","qri":"","structure":{"checksum":"QmQXfdYYubCvr9ePJtgABpp7N1fNsAnnywUJPYAJTvDhrn","errCount":0,"entries":7,"format":"json","length":19116,"qri":"","schema":{"type":"array"}},"transform":{"config":{"org":"qri-io","repo":"qri"},"path":"/","syntax":"starlark"},"Handle":"b5","Name":"test","PublicKey":"CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC/W17VPFX+pjyM1MHI1CsvIe+JYQX45MJNITTd7hZZDX2rRWVavGXhsccmVDGU6ubeN3t6ewcBlgCxvyewwKhmZKCAs3/0xNGKXK/YMyZpRVjTWw9yPU9gOzjd9GuNJtL7d1Hl7dPt9oECa7WBCh0W9u2IoHTda4g8B2mK92awLOZTjXeA7vbhKKX+QVHKDxEI0U2/ooLYJUVxEoHRc+DUYNPahX5qRgJ1ZDP4ep1RRRoZR+HjGhwgJP+IwnAnO5cRCWUbZvE1UBJUZDvYMqW3QvDp+TtXwqUWVvt69tp8EnlBgfyXU91A58IEQtLgZ7klOzdSEJDP+S8HIwhG/vbTAgMBAAE="}},{"Type":"","ID":"EDGI/fib_6","Value":{"commit":{"path":"/","signature":"dG6NoEFlQ9ILFjVDrecSDlUbDPRSiwK9kFQ3vjTh/4tpfgrT5EOw6eGDx75lklx0DWx51s3AC2Qqytll2JwwCB6SMVVl0I9qnZJ4XQVG+MX4hGeIJ4crGCSts85unDvmiQCfc4EVqPYZKLzaVqjXa43zv5mJPfRA2ktTew3VGkOcg8RmyU6e2XWrZpkILcZg1jt2apKs5qHslx4klKBVPtQIr53/U61OW4tzME9kz08FYrk2F7I5FHWy45W7VU8DpzCbhw6kxJXu2KYD1QstsZGCKH93sZY3agP4XGY15HeEOTib465LK6+nsoBtrsroQSOTBHzVgyUZACNom5SUvQ==","timestamp":"2018-05-23T19:50:03.307982846Z","title":"created dataset"},"meta":{"description":"test of starlark as a transformation language","qri":"md:0","title":"Fibonacci(6)"},"path":"/ipfs/QmS6jJSEJYxZvCeo8cZqzVa7Ybu9yNQeFYfNZAHxM4eyDK","qri":"","structure":{"checksum":"QmdhDDZTAWoifKCWwFrZqzKUyXM6rN7ThdP1u67rkeJvTj","errCount":0,"entries":6,"format":"cbor","length":7,"qri":"","schema":{"type":"array"}},"transform":{"syntax":"starlark"},"Handle":"EDGI","Name":"fib_6","PublicKey":"CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCmTFRx/6dKmoxje8AG+jFv94IcGUGnjrupa7XEr12J/c4ZLn3aPrD8F0tjRbstt1y/J+bO7Qb69DGiu2iSIqyE21nl2oex5+14jtxbupRq9jRTbpUHRj+y9I7uUDwl0E2FS1IQpBBfEGzDPIBVavxbhguC3O3XA7Aq7vea2lpJ1tWpr0GDRYSNmJAybkHS6k7dz1eVXFK+JE8FGFJi/AThQZKWRijvWFdlZvb8RyNFRHzpbr9fh38bRMTqhZpw/YGO5Ly8PNSiOOE4Y5cNUHLEYwG2/lpT4l53iKScsaOazlRkJ6NmkM1il7riCa55fcIAQZDtaAx+CT5ZKfmek4P5AgMBAAE="}}],"meta":{"code":200}}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(mockResponse)
	}))
	rc := regclient.NewClient(&regclient.Config{Location: server.URL})

	f, err := NewTestFactory(rc)
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
		sr, err := f.SearchRequests()
		if err != nil {
			t.Errorf("case %d, error creating dataset request: %s", i, err)
			continue
		}

		opt := &SearchOptions{
			IOStreams:      streams,
			Query:          c.query,
			Format:         c.format,
			SearchRequests: sr,
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

var textSearchResponse = `showing 3 results for 'test'
1   ramfox/test


2   b5/test
    /ipfs/QmPi5wrPsY4xPwy2oRr7NRZyfFxTeupfmnrVDubzoABLNP
    18 KBs, 7 entries, 0 errors

3   EDGI/fib_6
    Fibonacci(6)
    /ipfs/QmS6jJSEJYxZvCeo8cZqzVa7Ybu9yNQeFYfNZAHxM4eyDK
    7 bytes, 6 entries, 0 errors

`

var jsonSearchResponse = `[
  {
    "Type": "",
    "ID": "ramfox/test",
    "Value": null
  },
  {
    "Type": "",
    "ID": "b5/test",
    "Value": {
      "Handle": "b5",
      "Name": "test",
      "PublicKey": "CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC/W17VPFX+pjyM1MHI1CsvIe+JYQX45MJNITTd7hZZDX2rRWVavGXhsccmVDGU6ubeN3t6ewcBlgCxvyewwKhmZKCAs3/0xNGKXK/YMyZpRVjTWw9yPU9gOzjd9GuNJtL7d1Hl7dPt9oECa7WBCh0W9u2IoHTda4g8B2mK92awLOZTjXeA7vbhKKX+QVHKDxEI0U2/ooLYJUVxEoHRc+DUYNPahX5qRgJ1ZDP4ep1RRRoZR+HjGhwgJP+IwnAnO5cRCWUbZvE1UBJUZDvYMqW3QvDp+TtXwqUWVvt69tp8EnlBgfyXU91A58IEQtLgZ7klOzdSEJDP+S8HIwhG/vbTAgMBAAE=",
      "commit": {
        "path": "/",
        "signature": "JI/VSNqMuFGYVEwm3n8ZMjZmey+W2mhkD5if2337wDp+kaYfek9DntOyZiILXocW5JuOp48EqcsWf/BwhejQiYZ2utaIzR8VcMPo7u7c5nz2G6JTsoW+u9UUaKRVtl30jh6Kg1O2bnhYh9v4qW9VQgxOYfdhBl6zT4cYcjm1UkrblEe/wh494k9NziM5Bi2ATGRE2m71Lsf/TEDoNI549SebLQ1dsWXr1kM7lCeFqlDVjgbKQmGXowqcK/P9v+RBIRCnArnwFe/BQq4i1wmmnMEqpuUnfWR3xfJTE1DUMVaAid7U0jTWGVxROUdKk6mmTzlb1PiNdfruP+SFhjyQwQ==",
        "timestamp": "2018-05-25T13:44:54.692493401Z",
        "title": "created dataset"
      },
      "meta": {
        "citations": [
          {
            "url": "https://api.github.com/repos/qri-io/qri/releases"
          }
        ],
        "qri": "md:0"
      },
      "path": "/ipfs/QmPi5wrPsY4xPwy2oRr7NRZyfFxTeupfmnrVDubzoABLNP",
      "qri": "",
      "structure": {
        "checksum": "QmQXfdYYubCvr9ePJtgABpp7N1fNsAnnywUJPYAJTvDhrn",
        "entries": 7,
        "errCount": 0,
        "format": "json",
        "length": 19116,
        "qri": "",
        "schema": {
          "type": "array"
        }
      },
      "transform": {
        "config": {
          "org": "qri-io",
          "repo": "qri"
        },
        "path": "/",
        "syntax": "starlark"
      }
    }
  },
  {
    "Type": "",
    "ID": "EDGI/fib_6",
    "Value": {
      "Handle": "EDGI",
      "Name": "fib_6",
      "PublicKey": "CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCmTFRx/6dKmoxje8AG+jFv94IcGUGnjrupa7XEr12J/c4ZLn3aPrD8F0tjRbstt1y/J+bO7Qb69DGiu2iSIqyE21nl2oex5+14jtxbupRq9jRTbpUHRj+y9I7uUDwl0E2FS1IQpBBfEGzDPIBVavxbhguC3O3XA7Aq7vea2lpJ1tWpr0GDRYSNmJAybkHS6k7dz1eVXFK+JE8FGFJi/AThQZKWRijvWFdlZvb8RyNFRHzpbr9fh38bRMTqhZpw/YGO5Ly8PNSiOOE4Y5cNUHLEYwG2/lpT4l53iKScsaOazlRkJ6NmkM1il7riCa55fcIAQZDtaAx+CT5ZKfmek4P5AgMBAAE=",
      "commit": {
        "path": "/",
        "signature": "dG6NoEFlQ9ILFjVDrecSDlUbDPRSiwK9kFQ3vjTh/4tpfgrT5EOw6eGDx75lklx0DWx51s3AC2Qqytll2JwwCB6SMVVl0I9qnZJ4XQVG+MX4hGeIJ4crGCSts85unDvmiQCfc4EVqPYZKLzaVqjXa43zv5mJPfRA2ktTew3VGkOcg8RmyU6e2XWrZpkILcZg1jt2apKs5qHslx4klKBVPtQIr53/U61OW4tzME9kz08FYrk2F7I5FHWy45W7VU8DpzCbhw6kxJXu2KYD1QstsZGCKH93sZY3agP4XGY15HeEOTib465LK6+nsoBtrsroQSOTBHzVgyUZACNom5SUvQ==",
        "timestamp": "2018-05-23T19:50:03.307982846Z",
        "title": "created dataset"
      },
      "meta": {
        "description": "test of starlark as a transformation language",
        "qri": "md:0",
        "title": "Fibonacci(6)"
      },
      "path": "/ipfs/QmS6jJSEJYxZvCeo8cZqzVa7Ybu9yNQeFYfNZAHxM4eyDK",
      "qri": "",
      "structure": {
        "checksum": "QmdhDDZTAWoifKCWwFrZqzKUyXM6rN7ThdP1u67rkeJvTj",
        "entries": 6,
        "errCount": 0,
        "format": "cbor",
        "length": 7,
        "qri": "",
        "schema": {
          "type": "array"
        }
      },
      "transform": {
        "syntax": "starlark"
      }
    }
  }
]`
