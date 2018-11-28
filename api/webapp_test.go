package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/qri-io/qri/config"
)

func TestServeWebapp(t *testing.T) {

	node, teardown := newTestNode(t)
	defer teardown()
	cfg := config.DefaultConfigForTesting()
	cfg.Webapp.Enabled = false
	New(node, cfg).ServeWebapp()

	cfg.Webapp.EntrypointUpdateAddress = ""
	cfg.Webapp.Enabled = true
	s := New(node, cfg)
	go s.ServeWebapp()

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
