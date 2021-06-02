package run_test

import (
	"context"
	"testing"

	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/spec"
	"github.com/qri-io/qri/event"
)

func TestMemStore(t *testing.T) {
	ctx := context.Background()
	bus := event.NewBus(ctx)
	store := run.NewMemStore(bus)
	spec.AssertRunStore(t, store)
}
