package transform

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/dataset"
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
				{Type: event.ETTransformStart, Payload: event.TransformLifecycle{RunID: "three_step_success", StepCount: 3, Mode: "apply"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "setup", Mode: "apply"}},
				{Type: event.ETTransformPrint, Payload: event.TransformMessage{Msg: "oh, hello!"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "setup", Status: StatusSucceeded, Mode: "apply"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "download", Mode: "apply"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "download", Status: StatusSucceeded, Mode: "apply"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "transform", Mode: "apply"}},
				{Type: event.ETTransformDatasetPreview, Payload: threeStepDatasetPreview},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "transform", Status: StatusSucceeded, Mode: "apply"}},
				{Type: event.ETTransformStop, Payload: event.TransformLifecycle{Status: StatusSucceeded, Mode: "apply"}},
			},
		},

		{"one_step_error",
			&dataset.Transform{
				Steps: []*dataset.TransformStep{
					{Syntax: "starlark", Category: "setup", Script: `error("dang, it broke.")`},
				},
			},
			[]event.Event{
				{Type: event.ETTransformStart, Payload: event.TransformLifecycle{RunID: "one_step_error", StepCount: 1, Mode: "apply"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "setup", Mode: "apply"}},
				{Type: event.ETTransformError, Payload: event.TransformMessage{Lvl: event.TransformMsgLvlError, Msg: "Traceback (most recent call last):\n  .star:1:6: in <toplevel>\nError in error: transform error: \"dang, it broke.\"", Mode: "apply"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "setup", Status: StatusFailed, Mode: "apply"}},
				{Type: event.ETTransformStop, Payload: event.TransformLifecycle{Status: StatusFailed, Mode: "apply"}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := applyNoHistoryTransform(t, c.tf, c.name, "apply")
			compareEventLogs(t, c.expect, log)
		})
	}
}

func TestCommit(t *testing.T) {
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
				{Type: event.ETTransformStart, Payload: event.TransformLifecycle{RunID: "three_step_success", StepCount: 3, Mode: "commit"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "setup", Mode: "commit"}},
				{Type: event.ETTransformPrint, Payload: event.TransformMessage{Msg: "oh, hello!"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "setup", Status: StatusSucceeded, Mode: "commit"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "download", Mode: "commit"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "download", Status: StatusSucceeded, Mode: "commit"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "transform", Mode: "commit"}},
				{Type: event.ETTransformDatasetPreview, Payload: threeStepDatasetPreview},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "transform", Status: StatusSucceeded, Mode: "commit"}},
				{Type: event.ETTransformStop, Payload: event.TransformLifecycle{Status: StatusSucceeded, Mode: "commit"}},
			},
		},

		{"one_step_error",
			&dataset.Transform{
				Steps: []*dataset.TransformStep{
					{Syntax: "starlark", Category: "setup", Script: `error("dang, it broke.")`},
				},
			},
			[]event.Event{
				{Type: event.ETTransformStart, Payload: event.TransformLifecycle{RunID: "one_step_error", StepCount: 1, Mode: "commit"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "setup", Mode: "commit"}},
				{Type: event.ETTransformError, Payload: event.TransformMessage{Lvl: event.TransformMsgLvlError, Msg: "Traceback (most recent call last):\n  .star:1:6: in <toplevel>\nError in error: transform error: \"dang, it broke.\"", Mode: "commit"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "setup", Status: StatusFailed, Mode: "commit"}},
				{Type: event.ETTransformStop, Payload: event.TransformLifecycle{Status: StatusFailed, Mode: "commit"}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := applyNoHistoryTransform(t, c.tf, c.name, "commit")
			compareEventLogs(t, c.expect, log)
		})
	}
}

// run a transform script & capture the event log. transform runs against an
// empty dataset history
func applyNoHistoryTransform(t *testing.T, tf *dataset.Transform, runID, runMode string) []event.Event {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scriptOut := &bytes.Buffer{}
	loader := &noHistoryLoader{}
	target := &dataset.Dataset{Transform: tf}

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

	transformer := NewTransformer(ctx, loader, bus)
	if runMode == "apply" {
		if err := transformer.Apply(ctx, target, runID, false, scriptOut, nil); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := transformer.Commit(ctx, target, runID, false, scriptOut, nil); err != nil {
			t.Fatal(err)
		}
	}

	<-doneCh
	return log
}

type noHistoryLoader struct{}

// LoadDataset fails and returns that the reference has no history
func (l *noHistoryLoader) LoadDataset(ctx context.Context, ref string) (*dataset.Dataset, error) {
	return nil, dsref.ErrNoHistory
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
