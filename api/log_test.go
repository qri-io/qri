package api

import (
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	reporef "github.com/qri-io/qri/repo/ref"
)

func TestHistoryHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()
	inst := lib.NewInstanceFromConfigAndNode(config.DefaultConfigForTesting(), node)

	res := &reporef.DatasetRef{}
	p := &lib.SaveParams{
		Ref: "me/cities",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "Updated Title",
			},
		},
		Private: false,
	}
	if err := lib.NewDatasetMethods(inst).Save(p, res); err != nil {
		t.Fatalf("error writing dataset update: %s", err.Error())
	}

	h := NewLogHandlers(inst)

	logCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		// TODO (b5) - these currently break in CI b/c of timzone mismatching
		// we need to get timezones fixed for logbook
		// {"GET", "/history/me/cities", nil},
		// {"GET", "/history/me/cities/at/map/QmZrmGvTPMCkJYfqaagFZBUWuX5bkqSXu179eNnFfhCKze", nil},
		// {"GET", "/history/at/map/QmZrmGvTPMCkJYfqaagFZBUWuX5bkqSXu179eNnFfhCKze", nil},
		// {"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "log", h.LogHandler, logCases, true)
}
