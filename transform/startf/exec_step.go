package startf

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
	skyctx "github.com/qri-io/qri/transform/startf/context"
	skyds "github.com/qri-io/qri/transform/startf/ds"
	skyqri "github.com/qri-io/qri/transform/startf/qri"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

type StepRunner interface {
	RunStep(ctx context.Context, ds *dataset.Dataset, st *dataset.TransformStep) error
}

type stepRunner struct {
	runID        string
	starCtx      *skyctx.Context
	loadDataset  dsref.ParseResolveLoad
	repo         repo.Repo
	next         *dataset.Dataset
	prev         *dataset.Dataset
	skyqri       *skyqri.Module
	checkFunc    func(path ...string) error
	globals      starlark.StringDict
	bodyFile     qfs.File
	stderr       io.Writer
	moduleLoader ModuleLoader
	eventsCh     chan event.Event

	download starlark.Iterable
}

func NewStepRunner(eventsCh chan event.Event, runID string, prev, next *dataset.Dataset, opts ...func(o *ExecOpts)) StepRunner {
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

	// add error func to starlark environment
	starlark.Universe["error"] = starlark.NewBuiltin("error", Error)
	for key, val := range o.Globals {
		starlark.Universe[key] = val
	}

	// starCtx := skyctx.NewContext(o.Config, o.Secrets)
	starCtx := skyctx.NewContext(nil, o.Secrets)

	r := &stepRunner{
		starCtx:      starCtx,
		loadDataset:  o.DatasetLoader,
		repo:         o.Repo,
		skyqri:       skyqri.NewModule(o.Repo),
		prev:         prev,
		next:         next,
		checkFunc:    o.MutateFieldCheck,
		stderr:       o.ErrWriter,
		moduleLoader: o.ModuleLoader,
		eventsCh:     eventsCh,
	}

	return r
}

func (r *stepRunner) RunStep(ctx context.Context, ds *dataset.Dataset, st *dataset.TransformStep) error {
	thread := &starlark.Thread{
		Load: r.ModuleLoader,
		Print: func(thread *starlark.Thread, msg string) {
			// note we're ignoring a returned error here
			_, _ = r.stderr.Write([]byte(msg))
		},
	}

	globals, err := starlark.ExecFile(thread, fmt.Sprintf("%s.star", st.Name), strings.NewReader(st.Value), r.locals(ctx))
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			return fmt.Errorf(evalErr.Backtrace())
		}
		return err
	}

	if err := r.callStepFunc(globals, thread, st.Type); err != nil {
		return err
	}

	// funcs, err := t.specialFuncs()
	// if err != nil {
	// 	return err
	// }

	// for name, fn := range funcs {
	// 	eventsCh <- eventData{event.ETTransformStepStart, event.TransformStepLifecycle{Name: name}}
	// 	val, err := fn(t, thread, tfCtx)

	// 	if err != nil {
	// 		if evalErr, ok := err.(*starlark.EvalError); ok {
	// 			eventsCh <- eventData{event.ETError, event.TransformMessage{Msg: evalErr.Backtrace()}}
	// 			eventsCh <- eventData{event.ETTransformStepStop, event.TransformStepLifecycle{Name: name, Status: "failed"}}
	// 			return fmt.Errorf(evalErr.Backtrace())
	// 		}
	// 		eventsCh <- eventData{event.ETError, event.TransformMessage{Msg: err.Error()}}
	// 		eventsCh <- eventData{event.ETTransformStepStop, event.TransformStepLifecycle{Name: name, Status: "failed"}}
	// 		return err
	// 	}

	// 	eventsCh <- eventData{event.ETTransformStepStop, event.TransformStepLifecycle{Name: name, Status: "succeeded"}}
	// 	tfCtx.SetResult(name, val)
	// }

	// return err
	return nil
}

// ModuleLoader sums all loading assets to resolve a module name during transform execution
func (r *stepRunner) ModuleLoader(thread *starlark.Thread, module string) (dict starlark.StringDict, err error) {
	if r.moduleLoader == nil {
		return nil, fmt.Errorf("couldn't load module: %s", module)
	}

	return r.moduleLoader(thread, module)
}

func (r *stepRunner) callStepFunc(globals starlark.StringDict, thread *starlark.Thread, stepType string) error {
	log.Debugw("calling step function", "step", stepType, "globals", globals)
	if stepType == "setup" {
		r.globals = globals
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
		return r.callTransformFunc(thread, stepFunc)
	default:
		return fmt.Errorf("unrecognized starlark step type %q", stepType)
	}
}

// func (r *stepRunner) specialFuncs() (defined map[string]specialFunc, err error) {
// 	specialFuncs := map[string]specialFunc{
// 		"setup": r.callSetupFunc,
// 		"download": r.callDownloadFunc,
// 	}

// 	defined = map[string]specialFunc{}

// 	for name, fn := range specialFuncs {
// 		if _, err = t.globalFunc(name); err != nil {
// 			if err == ErrNotDefined {
// 				err = nil
// 				continue
// 			}
// 			return nil, err
// 		}
// 		defined[name] = fn
// 	}

// 	return
// }

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

func (r *stepRunner) callTransformFunc(thread *starlark.Thread, transform *starlark.Function) (err error) {
	d := skyds.NewDataset(r.prev, r.checkFunc)
	d.SetMutable(r.next)
	if _, err = starlark.Call(thread, transform, starlark.Tuple{d.Methods(), r.starCtx.Struct()}, nil); err != nil {
		return err
	}

	ds := r.next

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
	}
	r.eventsCh <- event.Event{Type: event.ETDataset, Payload: ds}
	// r.eventsCh <- event.Event{Type: event.ETTransformStepStop, Payload: event.TransformStepLifecycle{Name: "stepRunner", Status: "succeeded"}}

	return nil
}

func (r *stepRunner) locals(ctx context.Context) starlark.StringDict {
	log.Debugw("globals", "globals", r.globals)
	if r.globals == nil {
		return starlark.StringDict{
			"load_dataset": starlark.NewBuiltin("load_dataset", r.LoadDatasetFunc(ctx)),
			"print":        starlark.NewBuiltin("print", r.print),
		}
	}

	return r.globals
}

// LoadDataset implements the starlark load_dataset function
func (r *stepRunner) LoadDatasetFunc(ctx context.Context) func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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

		if r.next.Transform.Resources == nil {
			r.next.Transform.Resources = map[string]*dataset.TransformResource{}
		}

		r.next.Transform.Resources[ds.Path] = &dataset.TransformResource{
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
