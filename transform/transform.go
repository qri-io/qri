package transform

import (
	"context"
	"errors"
	"fmt"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/stepfile"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
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
	appCtx   context.Context
	loader   dsref.Loader
	fs       qfs.Filesystem
	pub      event.Publisher
	sizeInfo SizeInfo
	changes  map[string]struct{}
}

// SizeInfo is info about the size of the area that output is displayed on
type SizeInfo struct {
	OutputWidth  int
	OutputHeight int
}

// NewTransformer returns a new transformer
func NewTransformer(appCtx context.Context, fs qfs.Filesystem, loader dsref.Loader, pub event.Publisher, info SizeInfo) *Transformer {
	return &Transformer{
		appCtx:   appCtx,
		loader:   loader,
		fs:       fs,
		pub:      pub,
		sizeInfo: info,
	}
}

// Apply applies the transform script to a target dataset
func (t *Transformer) Apply(
	ctx context.Context,
	target *dataset.Dataset,
	runID string,
	wait bool,
	secrets map[string]string,
) error {
	return t.apply(ctx, "", target, runID, wait, secrets, RMApply)
}

// Commit applies the transform script to a target dataset, associating all
// events with the "commit" RunMode
func (t *Transformer) Commit(
	ctx context.Context,
	initID string,
	target *dataset.Dataset,
	runID string,
	wait bool,
	secrets map[string]string,
) error {
	return t.apply(ctx, initID, target, runID, wait, secrets, RMCommit)
}

func (t *Transformer) apply(
	ctx context.Context,
	initID string,
	target *dataset.Dataset,
	runID string,
	wait bool,
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

	ownerID := profile.IDFromCtx(ctx)

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
		startf.SetSecrets(secrets),
		startf.AddDatasetLoader(t.loader),
		startf.AddFilesystem(t.fs),
		startf.AddEventsChannel(eventsCh),
		startf.TrackChanges(t.changes),
		startf.SizeInfo(t.sizeInfo.OutputWidth, t.sizeInfo.OutputHeight),
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
			inTransformStep := false
			transformStepPayload := event.TransformStepLifecycle{}
			for {
				select {
				case e := <-eventsCh:
					t.pub.PublishID(ctx, e.Type, runID, e.Payload)
					if e.Type == event.ETTransformStop {
						receivedTransformStopEvt = true
					}
					if e.Type == event.ETTransformStepStart {
						inTransformStep = true
						ok := false
						transformStepPayload, ok = e.Payload.(event.TransformStepLifecycle)
						if !ok {
							log.Debug("transform.apply: event.ETTransformStepStart does not have the expected `event.TransformStepLifecycle` payload")
						}
					}
					if e.Type == event.ETTransformStop {
						inTransformStep = false
						transformStepPayload = event.TransformStepLifecycle{}
					}
				case <-ctx.Done():
					if !receivedTransformStopEvt {
						log.Warnw("context closed before transform stop event was sent", "runID", runID)

						ctx, cancel := context.WithCancel(t.appCtx)
						defer cancel()

						ctx = profile.AddIDToContext(ctx, ownerID)

						t.pub.PublishID(ctx, event.ETTransformError, runID, event.TransformMessage{
							Lvl:  event.TransformMsgLvlError,
							Msg:  "run canceled",
							Mode: runMode,
						})
						if inTransformStep {
							transformStepPayload.Status = "failed"
							t.pub.PublishID(ctx, event.ETTransformStepStop, runID, transformStepPayload)
						}
						t.pub.PublishID(ctx, event.ETTransformStop, runID, event.TransformLifecycle{
							InitID: initID,
							RunID:  runID,
							Mode:   runMode,
							Status: "failed",
						})
						err := t.pub.PublishID(ctx, event.ETTransformCanceled, runID, event.TransformLifecycle{
							InitID: initID,
							RunID:  runID,
							Mode:   runMode,
							Status: "failed",
						})
						if err != nil {
							log.Debugw("error publishing ETTransformCanceled", "err", err)
						}
					}
					return
				}
			}
		}()

		// "apply" runs are not expected to emit InitIDs in their
		// TransformLifecyle events
		eventsCh <- event.Event{Type: event.ETTransformStart, Payload: event.TransformLifecycle{RunID: runID, InitID: initID, StepCount: len(target.Transform.Steps), Mode: runMode}}

		var (
			runErr error
			status = StatusSucceeded
		)

		// Convert single-file transform scripts to steps
		if len(target.Transform.Steps) == 0 && target.Transform.ScriptFile() != nil {
			steps, err := stepfile.Read(target.Transform.ScriptFile())
			if err != nil {
				doneCh <- err
				return
			}
			for i := range steps {
				// assume steps are written in starlark. Only one syntax exists,
				// and stepfile doesn't detect syntax.
				steps[i].Syntax = "starlark"
			}
			target.Transform.Steps = steps
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
						Mode:     runMode,
					},
				}
				continue
			}

			eventsCh <- event.Event{
				Type: event.ETTransformStepStart,
				Payload: event.TransformStepLifecycle{
					Name:     step.Name,
					Category: step.Category,
					Mode:     runMode,
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
							Lvl:  event.TransformMsgLvlError,
							Msg:  runErr.Error(),
							Mode: runMode,
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
							Lvl:  event.TransformMsgLvlError,
							Msg:  fmt.Sprintf("unsupported transform syntax %q", step.Syntax),
							Mode: runMode,
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
					Mode:     runMode,
				},
			}
		}

		// warn user if commit wasn't called
		if status != StatusFailed && !stepRunner.CommitCalled() {
			eventsCh <- event.Event{
				Type: event.ETTransformPrint,
				Payload: event.TransformMessage{
					Lvl: event.TransformMsgLvlWarn,
					Msg: "this script did not call dataset.commit, no changes will be saved",
				},
			}
		}

		eventsCh <- event.Event{
			Type: event.ETTransformStop,
			Payload: event.TransformLifecycle{
				// "apply" runs are not expected to emit InitIDs
				// in their TransformLifecycle events
				InitID: initID,
				RunID:  runID,
				Mode:   runMode,
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
