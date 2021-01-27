package transform

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
)

func TestApply(t *testing.T) {
	tf := &dataset.Transform{
		Steps: []*dataset.TransformStep{
			{Syntax: "starlark", Category: "setup", Script: ""},
			{Syntax: "starlark", Category: "download", Script: "def download(ctx):\n\treturn"},
			{Syntax: "starlark", Category: "transform", Script: "def transform(ds, ctx):\n\tds.set_body([[1,2,3]])"},
		},
	}
	log := applyNoHistoryTransform(t, tf)

	expect := []event.Event{
		{Type: event.ETTransformStart, Payload: nil},
		{Type: event.ETTransformStepStart, Payload: event.TransformStepDetail{Category: "setup"}},
		{Type: event.ETTransformStepStop, Payload: event.TransformStepDetail{Category: "setup", Success: true}},
		{Type: event.ETTransformStepStart, Payload: event.TransformStepDetail{Category: "download"}},
		{Type: event.ETTransformStepStop, Payload: event.TransformStepDetail{Category: "download", Success: true}},
		{Type: event.ETTransformStepStart, Payload: event.TransformStepDetail{Category: "transform"}},
		{Type: event.ETTransformStepStop, Payload: event.TransformStepDetail{Category: "transform", Success: true}},
		{Type: event.ETTransformComplete, Payload: nil},
	}
	compareEventLogs(t, expect, log)
}

// run a transform script & capture the event log. transform runs against an
// empty dataset history
func applyNoHistoryTransform(t *testing.T, tf *dataset.Transform) []event.Event {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streams := ioes.NewDiscardIOStreams()
	scriptOut := &bytes.Buffer{}
	noHistoryLoader := func(ctx context.Context, refStr string) (*dataset.Dataset, error) {
		return nil, dsref.ErrNoHistory
	}
	target := &dataset.Dataset{Transform: tf}

	runID := NewRunID()
	bus := event.NewBus(ctx)
	log := []event.Event{}
	doneCh := make(chan struct{})
	bus.SubscribeID(func(ctx context.Context, e event.Event) error {

		log = append(log, e)
		switch e.Type {
		case event.ETTransformComplete:
			doneCh <- struct{}{}
		case event.ETTransformFailure:
			doneCh <- struct{}{}
		}
		return nil
	}, runID)

	if err := Apply(ctx, target, noHistoryLoader, runID, bus, false, streams, scriptOut, nil); err != nil {
		t.Fatal(err)
	}

	<-doneCh
	return log
}

// compareEventLogs asserts two event log slices are roughly equal,
// ignoring Timestamp & SessionID fields
func compareEventLogs(t *testing.T, expect, log []event.Event) {
	t.Helper()
	ignorePaths := func(p cmp.Path) bool {
		switch p.Last().String() {
		case ".Timestamp", ".SessionID":
			return true
		default:
			return false
		}
	}
	if diff := cmp.Diff(expect, log, cmp.FilterPath(ignorePaths, cmp.Ignore())); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
