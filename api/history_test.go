package api

import (
	"testing"

	"github.com/qri-io/dataset"

	"github.com/qri-io/qri/lib"
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

	h := NewHistoryHandlers(r)

	logCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/history/me/cities", nil},
		{"GET", "/history/me/cities/at/map/QmZrmGvTPMCkJYfqaagFZBUWuX5bkqSXu179eNnFfhCKze", nil},
		{"GET", "/history/at/map/QmZrmGvTPMCkJYfqaagFZBUWuX5bkqSXu179eNnFfhCKze", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "log", h.LogHandler, logCases)
}
