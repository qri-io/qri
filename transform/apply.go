package transform

import (
	"context"
	"errors"
	"fmt"
	"io"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
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

// Transformer holds long-lived values needed to apply transforms
type Transformer struct {
	AppCtx   context.Context
	LoadFunc dsref.ParseResolveLoad
	Pub      event.Publisher
}

// NewTransformer returns a new transformer
func NewTransformer(appCtx context.Context, loadFunc dsref.ParseResolveLoad, pub event.Publisher) *Transformer {
	return &Transformer{
		AppCtx:   appCtx,
		LoadFunc: loadFunc,
		Pub:      pub,
	}
}

// Apply applies the transform script to a target dataset
func (t *Transformer) Apply(
	ctx context.Context,
	target *dataset.Dataset,
	runID string,
	wait bool,
	scriptOut io.Writer,
	secrets map[string]string,
) error {
	log.Debugw("applying transform", "runID", runID, "wait", wait)

	var (
		head *dataset.Dataset
		err  error
	)

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
		head, err = t.LoadFunc(ctx, fmt.Sprintf("%s/%s", target.Peername, target.Name))
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
		startf.AddDatasetLoader(t.LoadFunc),
		startf.AddEventsChannel(eventsCh),
	}

	doneCh := make(chan error)

	// Run the transform asynchronously. If wait is true, the main routine will wait
	// until doneCh gets signaled.
	go func() {
		if !wait {
			// if we're running this script async, bind to the background context
			// note that we lose any values attached to the given context
			ctx = t.AppCtx
			doneCh <- nil
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Forward events from the events channel to the eventBus
		go func() {
			receivedTransformStopEvt := false
			for {
				select {
				case e := <-eventsCh:
					t.Pub.PublishID(ctx, e.Type, runID, e.Payload)
					if e.Type == event.ETTransformStop {
						receivedTransformStopEvt = true
					}
				case <-ctx.Done():
					if !receivedTransformStopEvt {
						log.Warnw("context closed before transform stop event was sent", "runID", runID)
					}
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
				runErr = stepRunner.RunStep(ctx, target, step)
				if runErr != nil {
					log.Debugw("error running transform step", "runID", runID, "index", i, "err", runErr)
					eventsCh <- event.Event{
						Type: event.ETTransformError,
						Payload: event.TransformMessage{
							Lvl: event.TransformMsgLvlError,
							Msg: runErr.Error(),
						},
					}
					status = StatusFailed
				}
				log.Debugw("ran starlark step", "runID", runID, "category", step.Category, "name", step.Name, "scriptLen", scriptLen(step))
			default:
				if step.Syntax == SyntaxQri && step.Name == "save" {
					log.Infow("ignoring qri save step", "runID", runID)
				} else {
					log.Debugw("skipping unknown step", "runID", runID, "syntax", step.Syntax, "name", step.Name)
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

// scriptLen returns the length of the script string, -1 if the script is not
// a string type
func scriptLen(step *dataset.TransformStep) int {
	if str, ok := step.Script.(string); ok {
		return len(str)
	}
	return -1
}
