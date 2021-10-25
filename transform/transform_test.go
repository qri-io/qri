package transform

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
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
					{Syntax: "starlark", Script: `print("oh, hello!")`},
					{Syntax: "starlark", Script: "ds = dataset.latest()"},
					{Syntax: "starlark", Script: "ds.body = [[1,2,3]]\ndataset.commit(ds)"},
				},
			},
			[]event.Event{
				{Type: event.ETTransformStart, Payload: event.TransformLifecycle{RunID: "three_step_success", StepCount: 3, Mode: "apply"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Mode: "apply"}},
				{Type: event.ETTransformPrint, Payload: event.TransformMessage{Msg: "oh, hello!"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Status: StatusSucceeded, Mode: "apply"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Mode: "apply"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Status: StatusSucceeded, Mode: "apply"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Mode: "apply"}},
				{Type: event.ETTransformDatasetPreview, Payload: threeStepDatasetPreview},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Status: StatusSucceeded, Mode: "apply"}},
				{Type: event.ETTransformStop, Payload: event.TransformLifecycle{RunID: "three_step_success", Status: StatusSucceeded, Mode: "apply"}},
			},
		},

		{"one_step_error",
			&dataset.Transform{
				Steps: []*dataset.TransformStep{
					{Syntax: "starlark", Script: `error("dang, it broke.")`},
				},
			},
			[]event.Event{
				{Type: event.ETTransformStart, Payload: event.TransformLifecycle{RunID: "one_step_error", StepCount: 1, Mode: "apply"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Mode: "apply"}},
				{Type: event.ETTransformError, Payload: event.TransformMessage{Lvl: event.TransformMsgLvlError, Msg: "Traceback (most recent call last):\n  .star:1:6: in <toplevel>\nError in error: transform error: \"dang, it broke.\"", Mode: "apply"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Status: StatusFailed, Mode: "apply"}},
				{Type: event.ETTransformStop, Payload: event.TransformLifecycle{RunID: "one_step_error", Status: StatusFailed, Mode: "apply"}},
			},
		},

		{"two_commit_calls_error",
			&dataset.Transform{
				Steps: []*dataset.TransformStep{
					{Syntax: "starlark", Script: "ds = dataset.latest()\ndataset.commit(ds)\ndataset.commit(ds)"},
				},
			},
			[]event.Event{
				{Type: event.ETTransformStart, Payload: event.TransformLifecycle{RunID: "two_commit_calls_error", StepCount: 1, Mode: "apply"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Mode: "apply"}},
				{Type: event.ETTransformDatasetPreview, Payload: &dataset.Dataset{
					Commit: &dataset.Commit{
						Message: "created dataset",
						Title:   "created dataset",
					},
					Transform: &dataset.Transform{
						Steps: []*dataset.TransformStep{
							{Syntax: "starlark", Script: "ds = dataset.latest()\ndataset.commit(ds)\ndataset.commit(ds)"},
						},
					}}},
				{Type: event.ETTransformError, Payload: event.TransformMessage{Lvl: event.TransformMsgLvlError, Msg: "Traceback (most recent call last):\n  .star:3:15: in <toplevel>\nError in commit: commit can only be called once in a transform script", Mode: "apply"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Status: StatusFailed, Mode: "apply"}},
				{Type: event.ETTransformStop, Payload: event.TransformLifecycle{RunID: "two_commit_calls_error", Status: StatusFailed, Mode: "apply"}},
			},
		},

		{"no_commit_calls_warning",
			&dataset.Transform{
				Steps: []*dataset.TransformStep{
					{Syntax: "starlark", Script: "ds = dataset.latest()"},
				},
			},
			[]event.Event{
				{Type: event.ETTransformStart, Payload: event.TransformLifecycle{RunID: "no_commit_calls_warning", StepCount: 1, Mode: "apply"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Mode: "apply"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Status: StatusSucceeded, Mode: "apply"}},
				{Type: event.ETTransformPrint, Payload: event.TransformMessage{Lvl: "warn", Msg: "this script did not call dataset.commit, no changes will be saved"}},
				{Type: event.ETTransformStop, Payload: event.TransformLifecycle{RunID: "no_commit_calls_warning", Status: StatusSucceeded, Mode: "apply"}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := applyNoHistoryTransform(t, "", c.tf, c.name, "apply")
			compareEventLogs(t, c.expect, log)
		})
	}
}

func TestCommit(t *testing.T) {
	cases := []struct {
		name   string
		initID string
		runID  string
		tf     *dataset.Transform
		expect []event.Event
	}{

		{"three_step_success",
			"three_step_success_init_id",
			"three_step_success_run_id",
			&dataset.Transform{
				Steps: []*dataset.TransformStep{
					{Syntax: "starlark", Script: `print("oh, hello!")`},
					{Syntax: "starlark", Script: "ds = dataset.latest()"},
					{Syntax: "starlark", Script: "ds.body = [[1,2,3]]\ndataset.commit(ds)"},
				},
			},
			[]event.Event{
				{Type: event.ETTransformStart, Payload: event.TransformLifecycle{InitID: "three_step_success_init_id", RunID: "three_step_success_run_id", StepCount: 3, Mode: "commit"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Mode: "commit"}},
				{Type: event.ETTransformPrint, Payload: event.TransformMessage{Msg: "oh, hello!"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Status: StatusSucceeded, Mode: "commit"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Mode: "commit"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Status: StatusSucceeded, Mode: "commit"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Mode: "commit"}},
				{Type: event.ETTransformDatasetPreview, Payload: threeStepDatasetPreview},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Status: StatusSucceeded, Mode: "commit"}},
				{Type: event.ETTransformStop, Payload: event.TransformLifecycle{InitID: "three_step_success_init_id", RunID: "three_step_success_run_id", Status: StatusSucceeded, Mode: "commit"}},
			},
		},

		{"one_step_error",
			"one_step_error_init_id",
			"one_step_error_run_id",
			&dataset.Transform{
				Steps: []*dataset.TransformStep{
					{Syntax: "starlark", Category: "setup", Script: `error("dang, it broke.")`},
				},
			},
			[]event.Event{
				{Type: event.ETTransformStart, Payload: event.TransformLifecycle{InitID: "one_step_error_init_id", RunID: "one_step_error_run_id", StepCount: 1, Mode: "commit"}},
				{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Category: "setup", Mode: "commit"}},
				{Type: event.ETTransformError, Payload: event.TransformMessage{Lvl: event.TransformMsgLvlError, Msg: "Traceback (most recent call last):\n  .star:1:6: in <toplevel>\nError in error: transform error: \"dang, it broke.\"", Mode: "commit"}},
				{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Category: "setup", Status: StatusFailed, Mode: "commit"}},
				{Type: event.ETTransformStop, Payload: event.TransformLifecycle{InitID: "one_step_error_init_id", RunID: "one_step_error_run_id", Status: StatusFailed, Mode: "commit"}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := applyNoHistoryTransform(t, c.initID, c.tf, c.runID, "commit")
			compareEventLogs(t, c.expect, log)
		})
	}
}

// run a transform script & capture the event log. transform runs against an
// empty dataset history
func applyNoHistoryTransform(t *testing.T, initID string, tf *dataset.Transform, runID, runMode string) []event.Event {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	fs := qfs.NewMemFS()
	transformer := NewTransformer(ctx, fs, loader, bus)
	if runMode == "apply" {
		if err := transformer.Apply(ctx, target, runID, false, nil); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := transformer.Commit(ctx, initID, target, runID, false, nil); err != nil {
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
	Commit: &dataset.Commit{
		Message: "created dataset",
		Title:   "created dataset",
	},
	Structure: &dataset.Structure{
		Format:       "csv",
		FormatConfig: map[string]interface{}{"lazyQuotes": true},
		Schema: map[string]interface{}{
			"items": map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"title": "field_1", "type": "integer"},
					map[string]interface{}{"title": "field_2", "type": "integer"},
					map[string]interface{}{"title": "field_3", "type": "integer"},
				},
				"type": "array",
			},
			"type": "array",
		},
		Length:  6,
		Entries: 1,
	},
	Transform: &dataset.Transform{
		Steps: []*dataset.TransformStep{
			{Syntax: "starlark", Script: `print("oh, hello!")`},
			{Syntax: "starlark", Script: "ds = dataset.latest()"},
			{Syntax: "starlark", Script: "ds.body = [[1,2,3]]\ndataset.commit(ds)"},
		},
	},
}

func scriptFile(t *testing.T, path string) qfs.File {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return qfs.NewMemfileBytes(path, data)
}

func TestApplyAssignsColumnsAndBody(t *testing.T) {
	ctx := context.Background()

	loader := &noHistoryLoader{}
	bus := event.NewBus(ctx)
	fs := qfs.NewMemFS()
	transformer := NewTransformer(ctx, fs, loader, bus)

	ds := &dataset.Dataset{Transform: &dataset.Transform{}}
	ds.Transform.SetScriptFile(scriptFile(t, "startf/testdata/csv_with_header.star"))
	err := transformer.Apply(ctx, ds, "myRunID", true, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Schema created from the csv header row
	actualBytes, err := json.Marshal(ds.Structure.Schema)
	if err != nil {
		t.Fatal(err)
	}
	actual := string(actualBytes)
	expect := `{"items":{"items":[{"title":"name","type":"string"},{"title":"sound","type":"string"}],"type":"array"},"type":"array"}`

	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}

	// Body contains just the rows without header
	actualBytes, err = ioutil.ReadAll(ds.BodyFile())
	if err != nil {
		t.Fatal(err)
	}
	actual = string(actualBytes)
	expect = "cat,meow\ndog,bark\n"

	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}

}
