package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
)

func TestServeWebapp(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	node, teardown := newTestNode(t)
	defer teardown()
	cfg := config.DefaultConfigForTesting()
	cfg.Webapp.Enabled = false
	// TODO (b5) - hack until tests have better instance-generation primitives
	inst := lib.NewInstanceFromConfigAndNode(cfg, node)
	New(inst).ServeWebapp(ctx)

	cfg.Webapp.EntrypointUpdateAddress = ""
	cfg.Webapp.Enabled = true
	s := New(inst)
	go s.ServeWebapp(ctx)

	url := fmt.Sprintf("http://localhost:%d", cfg.Webapp.Port)
	res, err := http.Get(url)
	if err != nil {
		t.Error(err.Error())
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
	if string(data) != webapptmpl {
		t.Errorf("base url should return webapp template")
	}

	_, err = http.Get(url + "/webapp/main.js")
	if err != nil {
		t.Error(err.Error())
	}

	// TODO - actually respond with something meaningful
}
