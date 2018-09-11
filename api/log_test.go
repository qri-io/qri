package api

import (
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

func TestHistoryHandlers(t *testing.T) {
	r, teardown := newTestRepo(t)
	defer teardown()

	res := &repo.DatasetRef{}
	p := &lib.SaveParams{
		Dataset: &dataset.DatasetPod{
			Peername: "me",
			Name:     "cities",
			Meta: &dataset.Meta{
				Title: "Updated Title",
			},
		},
		Private: false,
	}
	if err := lib.NewDatasetRequests(r, nil).Save(p, res); err != nil {
		t.Fatalf("error writing dataset update: %s", err.Error())
	}

	cfg := config.DefaultP2PForTesting()
	cfg.Enabled = false
	node, err := p2p.NewTestableQriNode(r, cfg)
	if err != nil {
		t.Fatal(err.Error())
	}

	h := NewLogHandlers(node.(*p2p.QriNode))

	logCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/history/me/cities", nil},
		{"GET", "/history/me/cities/at/map/QmZrmGvTPMCkJYfqaagFZBUWuX5bkqSXu179eNnFfhCKze", nil},
		{"GET", "/history/at/map/QmZrmGvTPMCkJYfqaagFZBUWuX5bkqSXu179eNnFfhCKze", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "log", h.LogHandler, logCases)
}
