package lib

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/automation/workflow"
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

func TestDeploy(t *testing.T) {
	tr := newTestRunner(t)
	bodyFile := qfs.NewMemfileBytes("body.csv", []byte("1,2,3\n4,5,6"))
	ds := &dataset.Dataset{
		Name:     "test",
		Peername: tr.MustOwner(t).Peername,
		Transform: &dataset.Transform{
			ScriptBytes: []byte(`
body = """a,b,c
1,2,3
4,5,6
"""
def transform(ds,ctx):
	ds.set_body(body, parse_as="csv")
`),
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
	bus := tr.Instance.Bus()
	handleDeploy := func(ctx context.Context, e event.Event) error {
		switch e.Type {
		case event.ETDeployEnd:
			payload, ok := e.Payload.(event.DeployEvent)
			if !ok {
				deployEnded <- "event.ETDeployEnd payload not of type event.DeployEvent"
			}
			deployEnded <- payload.Error
		}
		return nil
	}
	bus.SubscribeTypes(handleDeploy, event.ETDeployEnd)
	if err := tr.Instance.WithSource("local").Automation().Deploy(tr.Ctx, p); err != nil {
		t.Fatalf("deploy unexpected error: %s", err)
	}
	errMsg := <-done
	if errMsg != "" {
		t.Errorf(errMsg)
	}
}
