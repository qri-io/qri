package lib

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/event"
)

func TestApplyTransform(t *testing.T) {
	tr := newTestRunner(t)
	defer tr.Delete()

	// Save a dataset with a body
	_, err := tr.SaveWithParams(&SaveParams{
		Ref:      "me/cities_ds",
		BodyPath: "testdata/cities_2/body.csv",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Apply a transformation
	res, err := tr.ApplyWithParams(tr.Ctx, &ApplyParams{
		Ref: "me/cities_ds",
		Transform: &dataset.Transform{
			ScriptPath: "testdata/cities_2/add_city.star",
		},
		Wait: true,
	})
	if err != nil {
		t.Error(err)
	}

	output, err := json.Marshal(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	actual := string(output)
	expect := `[["toronto",50000000,55.5,false],["new york",8500000,44.4,true],["chicago",300000,44.4,true],["chatham",35000,65.25,true],["raleigh",250000,50.65,true],["tokyo",9200000,48.5,false]]`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("qri list (-want +got):\n%s", diff)
	}

	p := &ApplyParams{
		Wait: true,
		Transform: &dataset.Transform{
			Text: `
body = """a,b,c
1,2,3
4,5,6
"""
def transform(ds,ctx):
	ds.set_body(body, parse_as="csv")
`,
		},
	}
	res, err = tr.ApplyWithParams(tr.Ctx, p)
	if err != nil {
		t.Fatal(err)
	}

	expectBody := json.RawMessage(`[[1,2,3],[4,5,6]]`)

	if diff := cmp.Diff(expectBody, res.Body); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}

func TestApplyTransformValidationFailure(t *testing.T) {
	tr := newTestRunner(t)
	defer tr.Delete()

	params := ApplyParams{}
	_, err := tr.Instance.Automation().Apply(tr.Ctx, &params)
	if err == nil {
		t.Fatal("expected err but did not get one")
	}
	expectErr := "one or both of Reference, Transform are required"
	if err.Error() != expectErr {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err)
	}
}

func TestAutomation(t *testing.T) {
	tr := newTestRunner(t)
	bodyFile := qfs.NewMemfileBytes("body.csv", []byte("1,2,3\n4,5,6"))
	ds := &dataset.Dataset{
		Name:     "test",
		Peername: tr.MustOwner(t).Peername,
		Transform: &dataset.Transform{
			Steps: []*dataset.TransformStep{
				&dataset.TransformStep{
					Name:     "transform",
					Syntax:   "starlark",
					Category: "transform",
					Script: `
body = """a,b,c
1,2,3
4,5,6
"""
def transform(ds,ctx):
	ds.set_body(body, parse_as="csv")
`,
				},
			},
		},
	}
	ds.SetBodyFile(bodyFile)
	wf := &workflow.Workflow{
		OwnerID: tr.MustOwner(t).ID,
		Active:  true,
	}
	p := &DeployParams{
		Dataset:  ds,
		Workflow: wf,
		Run:      true,
	}
	deployEnded := make(chan string)
	done := make(chan string)
	go func() {
		select {
		case errMsg := <-deployEnded:
			done <- errMsg
		case <-time.After(200 * time.Millisecond):
			done <- "timeout occured before deploy finished"
		}
	}()

	// A successfully deployed workflow will send on the bus when it is finished
	bus := tr.Instance.Bus()
	handleDeploy := func(ctx context.Context, e event.Event) error {
		switch e.Type {
		case event.ETAutomationDeployEnd:
			payload, ok := e.Payload.(event.DeployEvent)
			if !ok {
				deployEnded <- "event.ETAutomationDeployEnd payload not of type event.DeployEvent"
			}
			wf.ID = workflow.ID(payload.WorkflowID)
			wf.InitID = payload.InitID
			deployEnded <- payload.Error
		}
		return nil
	}
	bus.SubscribeTypes(handleDeploy, event.ETAutomationDeployEnd)

	// The context we pass in will be cancelled as soon as the call to
	// deploy returns. But the operation should still complete successfully,
	// because deployed workflows run asynchronously.
	ctxCancelable, cancel := context.WithCancel(tr.Ctx)
	if err := tr.Instance.WithSource("local").Automation().Deploy(ctxCancelable, p); err != nil {
		t.Fatalf("deploy unexpected error: %s", err)
	}
	cancel()

	// Wait to make sure the workflow runs without error
	errMsg := <-done
	if errMsg != "" {
		t.Errorf(errMsg)
	}

	if wf.WorkflowID() == "" {
		t.Fatal("expected workflow ID in deploy event payload")
	}
	if wf.InitID == "" {
		t.Fatal("expected dataset ID in deploy event payload")
	}

	expectWF := wf.Copy()
	expectWF.Triggers = []map[string]interface{}{}

	gotWF, err := tr.Instance.WithSource("local").Automation().Workflow(tr.Ctx, &WorkflowParams{WorkflowID: wf.WorkflowID()})
	if err != nil {
		t.Fatal(err)
	}
	expectWF.Created = gotWF.Created
	if diff := cmp.Diff(expectWF, gotWF); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	gotWF, err = tr.Instance.WithSource("local").Automation().Workflow(tr.Ctx, &WorkflowParams{InitID: wf.InitID})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectWF, gotWF); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	gotEvent := event.WorkflowStoppedEvent{}
	runEnded := make(chan string)
	handleRun := func(ctx context.Context, e event.Event) error {
		if e.Type == event.ETAutomationWorkflowStopped {
			ok := false
			gotEvent, ok = e.Payload.(event.WorkflowStoppedEvent)
			if !ok {
				runEnded <- "event.ETAutomationDeployEnd payload not of type event.DeployEvent"
				return nil
			}
			runEnded <- ""
		}
		return nil
	}
	runID := "test_run_id"
	bus.SubscribeTypes(handleRun, event.ETAutomationWorkflowStopped)
	expectEvent := event.WorkflowStoppedEvent{
		InitID:     wf.InitID,
		OwnerID:    wf.OwnerID,
		WorkflowID: wf.WorkflowID(),
		RunID:      runID,
		Status:     string(run.RSSucceeded),
	}
	gotID, err := tr.Instance.WithSource("local").Automation().Run(tr.Ctx, &RunParams{WorkflowID: wf.WorkflowID(), RunID: runID})
	if !errors.Is(err, dsfs.ErrNoChanges) {
		t.Fatal(err)
	}
	if gotID != runID {
		t.Errorf("runID mismatch, expected %s, got %s", runID, gotID)
	}
	errMsg = <-runEnded
	if errMsg != "" {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectEvent, gotEvent); diff != "" {
		t.Errorf("workflow event mismatch (-want +got):\n%s", diff)
	}

	if err := tr.Instance.WithSource("local").Automation().Remove(tr.Ctx, &WorkflowParams{WorkflowID: wf.WorkflowID()}); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.Instance.WithSource("local").Automation().Workflow(tr.Ctx, &WorkflowParams{WorkflowID: wf.WorkflowID()}); !errors.Is(err, workflow.ErrNotFound) {
		t.Fatalf("error mismatch: expected %q, got %q", workflow.ErrNotFound, err)
	}
}

func TestRunParamsValidate(t *testing.T) {
	p := &RunParams{}
	if err := p.Validate(); err == nil {
		t.Fatalf("expected validation error for empty `RunParams`, got nil")
	}
	p.WorkflowID = "wfid"
	p.InitID = "initid"
	p.Ref = "ref"
	if err := p.Validate(); err == nil {
		t.Fatalf("expected validation error for `RunParams` with all fields non empty, got nil")
	}
}

func TestWorkflowParamsValidate(t *testing.T) {
	p := &WorkflowParams{}
	if err := p.Validate(); err == nil {
		t.Fatalf("expected validation error for empty `WorkflowParams`, got nil")
	}
	p.WorkflowID = "wfid"
	p.InitID = "initid"
	p.Ref = "ref"
	if err := p.Validate(); err == nil {
		t.Fatalf("expected validation error for `WorkflowParams` with all fields non empty, got nil")
	}
}
