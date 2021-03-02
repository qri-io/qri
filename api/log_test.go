package api

import (
	"context"
	"testing"

	"github.com/qri-io/dataset"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/lib"
)

func TestHistoryHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inst := lib.NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)

	p := &lib.SaveParams{
		Ref: "me/cities",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "Updated Title",
			},
		},
		Private: false,
	}
	_, err := lib.NewDatasetMethods(inst).Save(ctx, p)
	if err != nil {
		t.Fatalf("error writing dataset update: %s", err.Error())
	}

	h := NewLogHandlers(inst)

	logCases := []handlerTestCase{
		// TODO (b5) - these currently break in CI b/c of timzone mismatching
		// we need to get timezones fixed for logbook
		// {"GET", "/history/me/cities?local=true", nil},
		// {"GET", "/history/me/cities/at/map/QmZrmGvTPMCkJYfqaagFZBUWuX5bkqSXu179eNnFfhCKze?local=true", nil},
		// {"GET", "/history/at/map/QmZrmGvTPMCkJYfqaagFZBUWuX5bkqSXu179eNnFfhCKze?local=true", nil},
		// {"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "log", h.LogHandler, logCases, true)
}
