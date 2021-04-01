package lib

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qri-io/qri/config"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/registry/regclient"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestSearch(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(mockResponse)
	}))
	rc := regclient.NewClient(&regclient.Config{Location: server.URL})
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfig(), node)
	inst.registry = rc

	p := &SearchParams{"nuun", 0, 100}
	got, err := inst.Search().Search(ctx, p)
	if err != nil {
		t.Error(err)
	}

	if 1 != len(got) {
		t.Errorf("expected: %d results, got: %d", 1, len(got))
	}
}

var mockResponse = []byte(`{"data":[
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
],"meta":{"code":200}}`)
