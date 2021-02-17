package transform

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
)

func TestApply(t *testing.T) {
	cases := []struct {
		name   string
		tf     *dataset.Transform
		expect []event.Event
	}{

		{"three_step_success",
			&dataset.Transform{
				Steps: []*dataset.TransformStep{
					{Syntax: "starlark", Category: "setup", Script: `print("oh, hello!")`},
					{Syntax: "starlark", Category: "download", Script: "def download(ctx):\n\treturn"},
					{Syntax: "starlark", Category: "transform", Script: "def transform(ds, ctx):\n\tds.set_body([[1,2,3]])"},
				},
			},
			[]event.Event{
				{Type: event.ETTransformStart, Payload: event.TransformLifecycle{StepCount: 3}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "setup"}},
				{Type: event.ETTransformPrint, Payload: event.TransformMessage{Msg: "oh, hello!"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "setup", Status: StatusSucceeded}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "download"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "download", Status: StatusSucceeded}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "transform"}},
				{Type: event.ETTransformDatasetPreview, Payload: threeStepDatasetPreview},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "transform", Status: StatusSucceeded}},
				{Type: event.ETTransformStop, Payload: event.TransformLifecycle{Status: StatusSucceeded}},
			},
		},

		{"one_step_error",
			&dataset.Transform{
				Steps: []*dataset.TransformStep{
					{Syntax: "starlark", Category: "setup", Script: `error("dang, it broke.")`},
				},
			},
			[]event.Event{
				{Type: event.ETTransformStart, Payload: event.TransformLifecycle{StepCount: 1}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "setup"}},
				{Type: event.ETTransformError, Payload: event.TransformMessage{Lvl: event.TransformMsgLvlError, Msg: "Traceback (most recent call last):\n  .star:1:6: in <toplevel>\n  <builtin>: in error\nError: transform error: \"dang, it broke.\""}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "setup", Status: StatusFailed}},
				{Type: event.ETTransformStop, Payload: event.TransformLifecycle{Status: StatusFailed}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := applyNoHistoryTransform(t, c.tf)
			compareEventLogs(t, c.expect, log)
		})
	}
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
		case event.ETTransformStop:
			doneCh <- struct{}{}
		}
		return nil
	}, runID)

	if err := NewService(ctx).Apply(ctx, target, noHistoryLoader, runID, bus, false, streams, scriptOut, nil); err != nil {
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
	ignoreUnexported := cmpopts.IgnoreUnexported(
		dataset.Dataset{},
		dataset.Transform{},
	)
	if diff := cmp.Diff(expect, log, cmp.FilterPath(ignorePaths, cmp.Ignore()), ignoreUnexported); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

var threeStepDatasetPreview = &dataset.Dataset{
	Body: json.RawMessage(`[[1,2,3]]`),
	Structure: &dataset.Structure{
		Format: "json",
		Schema: map[string]interface{}{"type": "array"},
	},
	Transform: &dataset.Transform{
		Steps: []*dataset.TransformStep{
			{Syntax: "starlark", Category: "setup", Script: `print("oh, hello!")`},
			{Syntax: "starlark", Category: "download", Script: "def download(ctx):\n\treturn"},
			{Syntax: "starlark", Category: "transform", Script: "def transform(ds, ctx):\n\tds.set_body([[1,2,3]])"},
		},
	},
}
