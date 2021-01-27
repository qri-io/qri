package transform

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/transform/startf"
)

var log = golog.Logger("transform")

const (
	// SyntaxStarlark identifies steps & scripts written in starlark syntax
	// they're executed by the startf subpackage
	SyntaxStarlark = "starlark"
	// SyntaxQri is not currently in use. It's planned for deprecation & removal
	SyntaxQri = "qri"
)

const (
	// StatusWaiting is the canonical constant for "waiting" execution state
	StatusWaiting = "waiting"
	// StatusRunning is the canonical constant for "running" execution state
	StatusRunning = "running"
	// StatusSucceeded is the canonical constant for "succeeded" execution state
	StatusSucceeded = "succeeded"
	// StatusFailed is the canonical constant for "failed" execution state
	StatusFailed = "failed"
	// StatusSkipped is the canonical constant for "skipped" execution state
	StatusSkipped = "skipped"
)

// Apply applies the transform script to order to modify the changing dataset
func Apply(
	ctx context.Context,
	target *dataset.Dataset,
	loader dsref.ParseResolveLoad,
	runID string,
	pub event.Publisher,
	wait bool,
	str ioes.IOStreams,
	scriptOut io.Writer,
	secrets map[string]string,
) error {
	var (
		head *dataset.Dataset
		err  error
	)

	log.Debugw("applying transform", "runID", runID, "wait", wait)

	if target.Transform == nil {
		return errors.New("apply requires a transform component")
	}
	if len(target.Transform.Steps) == 0 && target.Transform.ScriptFile() == nil {
		return errors.New("apply requires either transform component with steps or a script file")
	}
	if runID == "" {
		return errors.New("apply requires a runID")
	}

	if target.Name != "" {
		head, err = loader(ctx, fmt.Sprintf("%s/%s", target.Peername, target.Name))
		if errors.Is(err, dsref.ErrRefNotFound) || errors.Is(err, dsref.ErrNoHistory) {
			// Dataset either does not exist yet, or has no history. Not an error
			head = &dataset.Dataset{}
			err = nil
		} else if err != nil {
			return err
		}
	}

	eventsCh := make(chan event.Event)

	// create a check func from a record of all the parts that the datasetPod is changing,
	// the startf package will use this function to ensure the same components aren't modified
	mutateCheck := startf.MutatedComponentsFunc(target)

	opts := []func(*startf.ExecOpts){
		startf.AddMutateFieldCheck(mutateCheck),
		startf.SetErrWriter(scriptOut),
		startf.SetSecrets(secrets),
		startf.AddDatasetLoader(loader),
		startf.AddEventsChannel(eventsCh),
	}

	doneCh := make(chan error)

	// Run the transform asynchronously. If wait is true, the main routine will wait
	// until doneCh gets signaled.
	go func() {
		if !wait {
			doneCh <- nil
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Forward events from the events channel to the eventBus
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

		eventsCh <- event.Event{Type: event.ETTransformStart, Payload: event.TransformLifecycle{StepCount: len(target.Transform.Steps)}}

		var (
			runErr error
			status = StatusSucceeded
		)

		// Single-file transform scripts do not have steps, should be executed all at once.
		if len(target.Transform.Steps) == 0 {
			runErr = startf.ExecScript(ctx, target, head, opts...)
			if runErr != nil {
				status = StatusFailed
				eventsCh <- event.Event{
					Type: event.ETTransformError,
					Payload: event.TransformMessage{
						Lvl: event.TransformMsgLvlError,
						Msg: runErr.Error(),
					},
				}
			}

			eventsCh <- event.Event{
				Type: event.ETTransformStop,
				Payload: event.TransformLifecycle{
					Status: status,
				},
			}
			doneCh <- runErr
			return
		}

		// Run each step using a StepRunner
		stepRunner := startf.NewStepRunner(head, opts...)
		for i, step := range target.Transform.Steps {
			// If the transform has failed at some step, emit skip events for remaining steps.
			if status != StatusSucceeded {
				eventsCh <- event.Event{
					Type: event.ETTransformStepSkip,
					Payload: event.TransformStepLifecycle{
						Name:     step.Name,
						Category: step.Category,
					},
				}
				continue
			}

			eventsCh <- event.Event{
				Type: event.ETTransformStepStart,
				Payload: event.TransformStepLifecycle{
					Name:     step.Name,
					Category: step.Category,
				},
			}

			switch step.Syntax {
			case SyntaxStarlark:
				log.Debugw("runnning starlark step", "step", step)
				runErr = stepRunner.RunStep(ctx, target, step)
				if runErr != nil {
					log.Debugw("running transform step", "index", i, "err", runErr)
					eventsCh <- event.Event{
						Type: event.ETTransformError,
						Payload: event.TransformMessage{
							Lvl: event.TransformMsgLvlError,
							Msg: runErr.Error(),
						},
					}
					status = StatusFailed
				}
			default:
				if step.Syntax == SyntaxQri && step.Name == "save" {
					log.Info("ignoring qri save step")
				} else {
					log.Debugw("skipping unknown step", "syntax", step.Syntax)
					eventsCh <- event.Event{
						Type: event.ETTransformError,
						Payload: event.TransformMessage{
							Lvl: event.TransformMsgLvlError,
							Msg: fmt.Sprintf("unsupported transform syntax %q", step.Syntax),
						},
					}
					status = StatusFailed
				}
			}

			eventsCh <- event.Event{
				Type: event.ETTransformStepStop,
				Payload: event.TransformStepLifecycle{
					Name:     step.Name,
					Category: step.Category,
					Status:   status,
				},
			}
		}

		eventsCh <- event.Event{
			Type: event.ETTransformStop,
			Payload: event.TransformLifecycle{
				Status: status,
			},
		}
		doneCh <- runErr
	}()

	err = <-doneCh
	return err
}

// NewRunID creates a run identifier
func NewRunID() string {
	return uuid.New().String()
}
