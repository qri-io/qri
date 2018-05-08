package api

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/beme/abide"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/test"
	"github.com/qri-io/registry"
	"github.com/qri-io/registry/regserver/handlers"
)

func TestProfileHandler(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// bump up log level to keep test output clean
	golog.SetLogLevel("qriapi", "error")
	defer golog.SetLogLevel("qriapi", "info")

	// use a test registry server
	registryServer := httptest.NewServer(handlers.NewRoutes(registry.NewProfiles()))

	// in order to have consistent responses
	// we need to artificially specify the timestamp
	// we use the dsfs.Timestamp func variable to override
	// the actual time
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r, err := test.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	core.Config = config.DefaultConfig()
	core.Config.Profile = test.ProfileConfig()
	core.Config.Registry.Location = registryServer.URL
	prevSaveConfig := core.SaveConfig
	core.SaveConfig = func() error {
		p, err := profile.NewProfile(core.Config.Profile)
		if err != nil {
			return err
		}

		r.SetProfile(p)
		return err
	}
	defer func() { core.SaveConfig = prevSaveConfig }()

	if err != nil {
		t.Error(err.Error())
		return
	}

	cases := []struct {
		method, endpoint string
		body             []byte
	}{
		{"OPTIONS", "/", nil},
		{"GET", "/", nil},
		{"POST", "/", []byte(`{"id": "","created": "0001-01-01T00:00:00Z","updated": "0001-01-01T00:00:00Z","peername": "","type": "peer","email": "test@email.com","name": "test name","description": "test description","homeUrl": "http://test.url","color": "default","thumb": "/","profile": "/","poster": "/","twitter": "test"}`)},
	}

	proh := NewProfileHandlers(r, false)

	for _, c := range cases {
		name := fmt.Sprintf("profile-%s-%s", c.method, c.endpoint)
		req := httptest.NewRequest(c.method, c.endpoint, bytes.NewBuffer(c.body))

		w := httptest.NewRecorder()
		proh.ProfileHandler(w, req)
		res := w.Result()
		abide.AssertHTTPResponse(t, strings.ToLower(name), res)
	}
}
