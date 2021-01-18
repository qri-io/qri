package transform

import (
	"context"
	"errors"
	"fmt"
	"io"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/transform/startf"
)

var log = golog.Logger("transform")

// Apply executes the transform script to order to modify the changing dataset
func Apply(
	ctx context.Context,
	ds *dataset.Dataset,
	loader dsref.ParseResolveLoad,
	pub event.Publisher,
	wait bool,
	str ioes.IOStreams,
	scriptOut io.Writer,
	secrets map[string]string,
) (string, error) {
	var (
		target = ds
		head   *dataset.Dataset
		runID  = NewRunID()
		doneCh = make(chan error)
		err    error
	)

	log.Debugw("applying transform", "runID", runID, "wait", wait)

	if target.Transform == nil || target.Transform.ScriptFile() == nil {
		log.Debugw("validating transform", "transform", target.Transform)
		return runID, errors.New("apply requires a transform component with a script file")
	}

	if ds.Name != "" {
		head, err = loader(ctx, fmt.Sprintf("%s/%s", ds.Peername, ds.Name))
		if errors.Is(err, dsref.ErrRefNotFound) || errors.Is(err, dsref.ErrNoHistory) {
			// Dataset either does not exist yet, or has no history. Not an error
			head = &dataset.Dataset{}
			err = nil
		} else if err != nil {
			log.Debugw("loading head dataset", "err", err)
			return runID, err
		}
	}

	// create a check func from a record of all the parts that the datasetPod is changing,
	// the startf package will use this function to ensure the same components aren't modified
	mutateCheck := startf.MutatedComponentsFunc(target)

	opts := []func(*startf.ExecOpts){
		startf.AddMutateFieldCheck(mutateCheck),
		startf.SetErrWriter(scriptOut),
		startf.SetSecrets(secrets),
		startf.AddDatasetLoader(loader),
	}

	eventsCh := make(chan event.Event)
	stepRunner := startf.NewStepRunner(eventsCh, runID, head, ds, opts...)

	go func() {
		if !wait {
			doneCh <- nil
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		go func() {
			for {
				select {
				case event := <-eventsCh:
					pub.PublishID(ctx, event.Type, runID, event.Payload)
				case <-ctx.Done():
					return
				}
			}
		}()

		eventsCh <- event.Event{Type: event.ETTransformStart, Payload: event.TransformLifecycle{RunID: runID}}

		hasFailedStep := false
		for i, step := range ds.Transform.Steps {
			if hasFailedStep {
				eventsCh <- event.Event{Type: event.ETTransformStepSkip, Payload: event.TransformStepLifecycle{Name: step.Name, Type: step.Type}}
				continue
			}

			eventsCh <- event.Event{Type: event.ETTransformStepStart, Payload: event.TransformStepLifecycle{Name: step.Name, Type: step.Type}}
			status := "succeeded"

			switch step.Syntax {
			case "starlark":
				log.Debugw("runnning starlark step", "step", step)
				if err := stepRunner.RunStep(ctx, ds, step); err != nil {
					log.Debugw("running transform step", "index", i, "err", err)
					eventsCh <- event.Event{Type: event.ETError, Payload: event.TransformMessage{Msg: err.Error()}}
					status = "failed"
				}
			case "qri":

			default:
				log.Debugw("skipping default step", "step", step)
				eventsCh <- event.Event{Type: event.ETError, Payload: event.TransformMessage{Msg: fmt.Sprintf("unsupported transform syntax %q", step.Syntax)}}
				status = "failed"
			}

			if status == "failed" {
				hasFailedStep = true
			}
			eventsCh <- event.Event{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Name: step.Name, Status: status}}
		}

		// if f := ds.BodyFile(); f != nil {
		// 	if ds.Structure == nil {
		// 		if err := base.InferStructure(ds); err != nil {
		// 			log.Debugw("inferring structure", "err", err)
		// 			eventsCh <- event.Event{Type: event.ETError, Payload: event.TransformMessage{Msg: err.Error()}}
		// 			eventsCh <- event.Event{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Name: "stepRunner", Status: "failed"}}
		// 			return
		// 		}
		// 	}
		// 	if err := base.InlineJSONBody(ds); err != nil {
		// 		log.Debugw("inlining resulting dataset JSON body", "err", err)
		// 	}
		// }
		// eventsCh <- event.Event{Type: event.ETDataset, Payload: ds}
		// eventsCh <- event.Event{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Name: "stepRunner", Status: "succeeded"}}

		// restore consumed script file
		// next.Transform.SetScriptFile(qfs.NewMemfileBytes("stepRunner.star", buf.Bytes()))

		tfStatus := "succeeded"
		if hasFailedStep {
			tfStatus = "failed"
		}
		eventsCh <- event.Event{Type: event.ETTransformStop, Payload: event.TransformLifecycle{RunID: runID, Status: tfStatus}}

		doneCh <- err
	}()

	err = <-doneCh
	return runID, err
}
