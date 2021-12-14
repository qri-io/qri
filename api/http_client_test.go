package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/base/dsfs"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/lib"
	qhttp "github.com/qri-io/qri/lib/http"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo/test"
)

func TestHTTPClient(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prevXformVer := APIVersion
	APIVersion = "test_version"
	defer func() {
		APIVersion = prevXformVer
	}()

	// bump up log level to keep test output clean
	golog.SetLogLevel("qriapi", "error")
	defer golog.SetLogLevel("qriapi", "info")

	// to keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r, err := test.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}

	// Cannot use TestRunner because we need to set cfg.API.ReadOnly.
	// TODO(dlong): Add a testRunner call trace that does this correctly.
	cfg := testcfg.DefaultConfigForTesting()

	node, err := p2p.NewQriNode(r, cfg.P2P, event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	// TODO (b5) - hack until tests have better instance-generation primitives
	inst := lib.NewInstanceFromConfigAndNode(ctx, cfg, node)
	s := New(inst)

	server := httptest.NewServer(NewServerRoutes(s))
	sURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err.Error())
	}

	httpClient, err := qhttp.NewClient(cfg.API.Address)
	if err != nil {
		t.Fatal(err.Error())
	}

	// override with test URI
	httpClient.Address = sURL.Host
	httpClient.Protocol = "http"

	if err = httpClient.CallRaw(ctx, AEHome, "", nil, &bytes.Buffer{}); err != nil {
		t.Fatal(err.Error())
	}

	res := []dsref.VersionInfo{}
	p := lib.CollectionListParams{}
	err = httpClient.CallMethod(ctx, qhttp.AEList, http.MethodPost, "", p, &res)
	if err != nil {
		t.Fatal(err.Error())
	}

	expectBytes, err := ioutil.ReadFile("testdata/http_client/list.json")
	if err != nil {
		t.Fatalf("error reading expected bytes: %s", err)
	}
	var expect []dsref.VersionInfo
	if err := json.Unmarshal(expectBytes, &expect); err != nil {
		t.Fatal(err)
	}

	t.Skip("TODO(b5): collection update has broken this contract. fix both test & collection implementation")
	if diff := cmp.Diff(expect, res); diff != "" {
		t.Errorf("byte mismatch (-want +got):\n%s", diff)
	}
}
