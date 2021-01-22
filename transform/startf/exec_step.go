package startf

import (
	"context"
	"fmt"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	skyctx "github.com/qri-io/qri/transform/startf/context"
	skyds "github.com/qri-io/qri/transform/startf/ds"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

type StepRunner interface {
	RunStep(ctx context.Context, ds *dataset.Dataset, st *dataset.TransformStep) error
}

type stepRunner struct {
	runID       string
	starCtx     *skyctx.Context
	loadDataset dsref.ParseResolveLoad
	prev        *dataset.Dataset
	checkFunc   func(path ...string) error
	globals     starlark.StringDict
	bodyFile    qfs.File
	eventsCh    chan event.Event
	thread      *starlark.Thread

	download starlark.Iterable
}

func NewStepRunner(eventsCh chan event.Event, runID string, prev *dataset.Dataset, opts ...func(o *ExecOpts)) StepRunner {
	o := &ExecOpts{}
	DefaultExecOpts(o)
	for _, opt := range opts {
		opt(o)
	}

	// hoist execution settings to resolve package settings
	resolve.AllowFloat = o.AllowFloat
	resolve.AllowSet = o.AllowSet
	resolve.AllowLambda = o.AllowLambda
	resolve.AllowNestedDef = o.AllowNestedDef
	resolve.LoadBindsGlobally = true

	// add error func to starlark environment
	starlark.Universe["error"] = starlark.NewBuiltin("error", Error)
	for key, val := range o.Globals {
		starlark.Universe[key] = val
	}

	thread := &starlark.Thread{
		Load: o.ModuleLoader,
		Print: func(thread *starlark.Thread, msg string) {
			// note we're ignoring a returned error here
			_, _ = o.ErrWriter.Write([]byte(msg))
		},
	}

	// starCtx := skyctx.NewContext(o.Config, o.Secrets)
	starCtx := skyctx.NewContext(nil, o.Secrets)

	r := &stepRunner{
		starCtx:     starCtx,
		loadDataset: o.DatasetLoader,
		prev:        prev,
		checkFunc:   o.MutateFieldCheck,
		eventsCh:    eventsCh,
		thread:      thread,
		globals:     starlark.StringDict{},
	}

	return r
}

func (r *stepRunner) RunStep(ctx context.Context, ds *dataset.Dataset, st *dataset.TransformStep) error {
	r.globals["print"] = starlark.NewBuiltin("print", r.print)
	r.globals["load_dataset"] = starlark.NewBuiltin("load_dataset", r.LoadDatasetFunc(ctx, ds))

	script, ok := st.Script.(string)
	if !ok {
		return fmt.Errorf("starlark step Script must be a string. got %T", st.Script)
	}

	globals, err := starlark.ExecFile(r.thread, fmt.Sprintf("%s.star", st.Name), strings.NewReader(script), r.globals)
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			return fmt.Errorf(evalErr.Backtrace())
		}
		return err
	}

	for key, val := range globals {
		r.globals[key] = val
	}

	if err := r.callStepFunc(r.thread, st.Category, ds); err != nil {
		return err
	}

	return nil
}

func (r *stepRunner) callStepFunc(thread *starlark.Thread, stepType string, ds *dataset.Dataset) error {
	if stepType == "setup" {
		return nil
	}

	stepFunc, err := r.globalFunc(stepType)
	if err != nil {
		return err
	}

	switch stepType {
	case "download":
		return r.callDownloadFunc(thread, stepFunc)
	case "transform":
		return r.callTransformFunc(thread, stepFunc, ds)
	default:
		return fmt.Errorf("unrecognized starlark step type %q", stepType)
	}
}

// globalFunc checks if a global function is defined
func (r *stepRunner) globalFunc(name string) (fn *starlark.Function, err error) {
	x, ok := r.globals[name]
	if !ok {
		return fn, ErrNotDefined
	}
	if x.Type() != "function" {
		return fn, fmt.Errorf("%q is not a function", name)
	}
	return x.(*starlark.Function), nil
}

type specialFunc func(t *transform, thread *starlark.Thread, ctx *skyctx.Context) (result starlark.Value, err error)

func (r *stepRunner) callDownloadFunc(thread *starlark.Thread, download *starlark.Function) (err error) {
	httpGuard.EnableNtwk()
	defer httpGuard.DisableNtwk()

	val, err := starlark.Call(thread, download, starlark.Tuple{r.starCtx.Struct()}, nil)
	if err != nil {
		return err
	}

	r.starCtx.SetResult("download", val)
	return nil
}

func (r *stepRunner) callTransformFunc(thread *starlark.Thread, transform *starlark.Function, ds *dataset.Dataset) (err error) {
	d := skyds.NewDataset(r.prev, r.checkFunc)
	d.SetMutable(ds)
	if _, err = starlark.Call(thread, transform, starlark.Tuple{d.Methods(), r.starCtx.Struct()}, nil); err != nil {
		return err
	}

	if f := ds.BodyFile(); f != nil {
		if ds.Structure == nil {
			if err := base.InferStructure(ds); err != nil {
				log.Debugw("inferring structure", "err", err)
				return err
			}
		}
		if err := base.InlineJSONBody(ds); err != nil {
			log.Debugw("inlining resulting dataset JSON body", "err", err)
		}
		ds.SetBodyFile(qfs.NewMemfileBytes("body.json", ds.BodyBytes))
	}
	r.eventsCh <- event.Event{Type: event.ETDataset, Payload: ds}
	// r.eventsCh <- event.Event{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Name: "stepRunner", Status: "succeeded"}}

	return nil
}

// LoadDataset implements the starlark load_dataset function
func (r *stepRunner) LoadDatasetFunc(ctx context.Context, target *dataset.Dataset) func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var refstr starlark.String
		if err := starlark.UnpackArgs("load_dataset", args, kwargs, "ref", &refstr); err != nil {
			return starlark.None, err
		}

		if r.loadDataset == nil {
			return nil, fmt.Errorf("load_dataset function is not enabled")
		}

		ds, err := r.loadDataset(ctx, refstr.GoString())
		if err != nil {
			return starlark.None, err
		}

		if target.Transform.Resources == nil {
			target.Transform.Resources = map[string]*dataset.TransformResource{}
		}

		target.Transform.Resources[ds.Path] = &dataset.TransformResource{
			// TODO(b5) - this should be a method on dataset.Dataset
			// we should add an ID field to dataset, set that to the InitID, and
			// add fields to dataset.TransformResource that effectively make it the
			// same data structure as dsref.Ref
			Path: fmt.Sprintf("%s/%s@%s", ds.Peername, ds.Name, ds.Path),
		}

		return skyds.NewDataset(ds, nil).Methods(), nil
	}
}

func (r *stepRunner) print(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var message starlark.String
	if err := starlark.UnpackArgs("print", args, kwargs, "message", &message); err != nil {
		return starlark.None, err
	}

	r.eventsCh <- event.Event{Type: event.ETPrint, Payload: event.TransformMessage{Msg: message.GoString()}}

	return starlark.None, nil
}
