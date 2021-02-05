package api

import (
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
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/lib"
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
	cfg := config.DefaultConfigForTesting()

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

	httpClient, err := lib.NewHTTPClient(cfg.API.Address)
	if err != nil {
		t.Fatal(err.Error())
	}

	// override with test URI
	httpClient.Address = sURL.Host
	httpClient.Protocol = "http"

	err = httpClient.Call(ctx, lib.AEHome, nil, &map[string]interface{}{})
	if err != nil {
		t.Fatal(err.Error())
	}

	res := []dsref.VersionInfo{}
	err = httpClient.CallMethod(ctx, lib.AEList, http.MethodGet, nil, &res)
	if err != nil {
		t.Fatal(err.Error())
	}

	body, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err.Error())
	}
	// Compare the API response to the expected zip file
	expectBytes, err := ioutil.ReadFile("testdata/http_client/list.json")
	if err != nil {
		t.Fatalf("error reading expected bytes: %s", err)
	}
	if diff := cmp.Diff(string(expectBytes), string(body)); diff != "" {
		t.Errorf("byte mismatch (-want +got):\n%s", diff)
	}
}
