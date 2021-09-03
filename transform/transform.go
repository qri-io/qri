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

const (
	// RMApply indicates the transform was executed as an "apply"
	// meaning, the transform was run with no intension to save the
	// output dataset
	RMApply = "apply"
	// RMCommit indicates the transform was executed as a "commit"
	// meaning, the transform was run with the intension to save the
	// output dataset
	RMCommit = "commit"
)

// Transformer holds dependencies needed for applying a transform
type Transformer struct {
	appCtx  context.Context
	loader  dsref.Loader
	pub     event.Publisher
	changes map[string]struct{}
}

// NewTransformer returns a new transformer
func NewTransformer(appCtx context.Context, loader dsref.Loader, pub event.Publisher) *Transformer {
	return &Transformer{
		appCtx: appCtx,
		loader: loader,
		pub:    pub,
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
	return t.apply(ctx, target, runID, wait, scriptOut, secrets, RMApply)
}

// Commit applies the transform script to a target dataset, associating all
// events with the "commit" RunMode
func (t *Transformer) Commit(
	ctx context.Context,
	target *dataset.Dataset,
	runID string,
	wait bool,
	scriptOut io.Writer,
	secrets map[string]string,
) error {
	return t.apply(ctx, target, runID, wait, scriptOut, secrets, RMCommit)
}

func (t *Transformer) apply(
	ctx context.Context,
	target *dataset.Dataset,
	runID string,
	wait bool,
	scriptOut io.Writer,
	secrets map[string]string,
	runMode string,
) error {
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
		head, err := t.loader.LoadDataset(ctx, fmt.Sprintf("%s/%s", target.Peername, target.Name))
		if errors.Is(err, dsref.ErrRefNotFound) || errors.Is(err, dsref.ErrNoHistory) {
			// Dataset either does not exist yet, or has no history. Not an error
			head = &dataset.Dataset{}
			err = nil
		} else if err != nil {
			return err
		}

		head.DropTransientValues()
		head.DropDerivedValues()
		head.ID = ""
		head.Commit = nil
		// Assign target to head first, to copy values being assigned to
		// the target dataset, such as manual changes (added with the --file
		// command-line flag) and the transform itself
		head.Assign(target)
		// Then assign back to target, so that we end up using the same object
		// in memory. This sequence of assignments is basically doing:
		// target.AssignFieldsThatAreNotAlreadySetFrom(head)
		target.Assign(head)
	}

	t.changes = make(map[string]struct{})
	eventsCh := make(chan event.Event)

	opts := []func(*startf.ExecOpts){
		startf.SetErrWriter(scriptOut),
		startf.SetSecrets(secrets),
		startf.AddDatasetLoader(t.loader),
		startf.AddEventsChannel(eventsCh),
		startf.TrackChanges(t.changes),
	}

	doneCh := make(chan error)

	// Run the transform asynchronously. If wait is true, the main routine will wait
	// until doneCh gets signaled.
	go func() {
		if !wait {
			// if we're running this script async, bind to the background context
			// note that we lose any values attached to the given context
			ctx = t.appCtx
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
					t.pub.PublishID(ctx, e.Type, runID, e.Payload)
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
			runErr = startf.ExecScript(ctx, target, opts...)
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
		stepRunner := startf.NewStepRunner(target, opts...)
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

	return <-doneCh
}

// Changes returns which components were changed by the most recent application
func (t *Transformer) Changes() map[string]struct{} {
	return t.changes
}

// scriptLen returns the length of the script string, -1 if the script is not
// a string type
func scriptLen(step *dataset.TransformStep) int {
	if str, ok := step.Script.(string); ok {
		return len(str)
	}
	return -1
}
