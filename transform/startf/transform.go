package startf

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/preview"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
	starctx "github.com/qri-io/qri/transform/startf/context"
	stards "github.com/qri-io/qri/transform/startf/ds"
	"github.com/qri-io/qri/version"
	"github.com/qri-io/starlib"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

var (
	// Version is the version of qri that this transform was run with
	Version = version.Version
	// ErrNotDefined is for when a starlark value is not defined or does not exist
	ErrNotDefined = fmt.Errorf("not defined")
)

// ExecOpts defines options for execution
type ExecOpts struct {
	// loader for loading datasets
	DatasetLoader dsref.Loader
	// supply a repo to make the 'qri' module available in starlark
	Repo repo.Repo
	// allow floating-point numbers
	AllowFloat bool
	// allow set data type
	AllowSet bool
	// allow lambda expressions
	AllowLambda bool
	// allow nested def statements
	AllowNestedDef bool
	// passed-in secrets (eg: API keys)
	Secrets map[string]interface{}
	// global values to pass for script execution
	Globals starlark.StringDict
	// provide a writer to record script "stderr" output to
	ErrWriter io.Writer
	// starlark module loader function
	ModuleLoader ModuleLoader
	// channel to send events on
	EventsCh chan event.Event
	// map containing components that have been changed
	ChangeSet map[string]struct{}
}

// AddDatasetLoader is required to enable the load_dataset starlark builtin
func AddDatasetLoader(loader dsref.Loader) func(o *ExecOpts) {
	return func(o *ExecOpts) {
		o.DatasetLoader = loader
	}
}

// AddQriRepo adds a qri repo to execution options, providing scripted access
// to assets within the respoitory
func AddQriRepo(repo repo.Repo) func(o *ExecOpts) {
	return func(o *ExecOpts) {
		o.Repo = repo
	}
}

// AddEventsChannel sets an event channel to send events on
func AddEventsChannel(eventsCh chan event.Event) func(o *ExecOpts) {
	return func(o *ExecOpts) {
		o.EventsCh = eventsCh
	}
}

// SetSecrets assigns environment secret key-value pairs for script execution
func SetSecrets(secrets map[string]string) func(o *ExecOpts) {
	return func(o *ExecOpts) {
		if secrets != nil {
			// convert to map[string]interface{}, which the lower-level startf supports
			// until we're sure map[string]string is going to work in the majority of use cases
			s := map[string]interface{}{}
			for key, val := range secrets {
				s[key] = val
			}
			o.Secrets = s
		}
	}
}

// TrackChanges retains a map that tracks changes to dataset components
func TrackChanges(changes map[string]struct{}) func(o *ExecOpts) {
	return func(o *ExecOpts) {
		o.ChangeSet = changes
	}
}

// DefaultExecOpts applies default options to an ExecOpts pointer
func DefaultExecOpts(o *ExecOpts) {
	o.AllowFloat = true
	o.AllowSet = true
	o.AllowLambda = true
	o.Globals = starlark.StringDict{}
	o.ErrWriter = ioutil.Discard
	o.ModuleLoader = DefaultModuleLoader
}

// StepRunner is able to run individual transform steps
type StepRunner struct {
	starCtx   *starctx.Context
	dsLoader  dsref.Loader
	target    *dataset.Dataset
	globals   starlark.StringDict
	eventsCh  chan event.Event
	thread    *starlark.Thread
	changeSet map[string]struct{}
}

// NewStepRunner returns a new StepRunner for the given dataset
func NewStepRunner(target *dataset.Dataset, opts ...func(o *ExecOpts)) *StepRunner {
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

	thread := &starlark.Thread{Load: o.ModuleLoader}

	starCtx := starctx.NewContext(nil, o.Secrets)

	r := &StepRunner{
		starCtx:   starCtx,
		dsLoader:  o.DatasetLoader,
		eventsCh:  o.EventsCh,
		target:    target,
		thread:    thread,
		globals:   starlark.StringDict{},
		changeSet: o.ChangeSet,
	}

	return r
}

// RunStep runs the single transform step using the dataset
func (r *StepRunner) RunStep(ctx context.Context, ds *dataset.Dataset, st *dataset.TransformStep) error {
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

	if err := r.callStepFunc(ctx, r.thread, st.Category, ds); err != nil {
		return err
	}

	return nil
}

func (r *StepRunner) callStepFunc(ctx context.Context, thread *starlark.Thread, stepType string, ds *dataset.Dataset) error {
	// a step of type "setup" skips any global calls
	if stepType == "setup" {
		return nil
	}

	// call download if defined
	if dlFunc, err := r.globalFunc("download"); err == nil {
		if err := r.callDownloadFunc(thread, dlFunc); err != nil {
			return err
		}
	}

	// call transform if defined
	if tfFunc, err := r.globalFunc("transform"); err == nil {
		if err := r.callTransformFunc(ctx, thread, tfFunc, ds); err != nil {
			return err
		}
	}

	return nil
}

// globalFunc checks if a global function is defined
func (r *StepRunner) globalFunc(name string) (fn *starlark.Function, err error) {
	x, ok := r.globals[name]
	if !ok {
		return fn, ErrNotDefined
	}
	if x.Type() != "function" {
		return fn, fmt.Errorf("%q is not a function", name)
	}
	return x.(*starlark.Function), nil
}

func (r *StepRunner) callDownloadFunc(thread *starlark.Thread, download *starlark.Function) (err error) {
	httpGuard.EnableNtwk()
	defer httpGuard.DisableNtwk()

	val, err := starlark.Call(thread, download, starlark.Tuple{r.starCtx.Struct()}, nil)
	if err != nil {
		return err
	}

	r.starCtx.SetResult("download", val)
	return nil
}

func (r *StepRunner) callTransformFunc(ctx context.Context, thread *starlark.Thread, transform *starlark.Function, ds *dataset.Dataset) (err error) {
	d := stards.NewDataset(r.target)
	if _, err = starlark.Call(thread, transform, starlark.Tuple{d, r.starCtx.Struct()}, nil); err != nil {
		return err
	}

	// Which components were changed
	if r.changeSet != nil {
		changes := d.Changes()
		for comp := range changes {
			r.changeSet[comp] = changes[comp]
		}
	}

	if r.eventsCh != nil {
		pview, err := preview.Create(ctx, ds)
		if err != nil {
			return err
		}
		r.eventsCh <- event.Event{Type: event.ETTransformDatasetPreview, Payload: pview}
	}

	return nil
}

// LoadDatasetFunc returns an implementation of the starlark load_dataset function
func (r *StepRunner) LoadDatasetFunc(ctx context.Context, target *dataset.Dataset) func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var refstr starlark.String
		if err := starlark.UnpackArgs("load_dataset", args, kwargs, "ref", &refstr); err != nil {
			return starlark.None, err
		}

		if r.dsLoader == nil {
			return nil, fmt.Errorf("load_dataset function is not enabled")
		}

		ds, err := r.dsLoader.LoadDataset(ctx, refstr.GoString())
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

		return stards.NewDataset(ds), nil
	}
}

func (r *StepRunner) print(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var message starlark.String
	if err := starlark.UnpackArgs("print", args, kwargs, "message", &message); err != nil {
		return starlark.None, err
	}
	if r.eventsCh != nil {
		r.eventsCh <- event.Event{
			Type: event.ETTransformPrint,
			Payload: event.TransformMessage{
				Msg: message.GoString(),
			},
		}
	}
	return starlark.None, nil
}

// ModuleLoader is a function that can load starlark modules
type ModuleLoader func(thread *starlark.Thread, module string) (starlark.StringDict, error)

// DefaultModuleLoader loads starlib modules
var DefaultModuleLoader = func(thread *starlark.Thread, module string) (dict starlark.StringDict, err error) {
	return starlib.Loader(thread, module)
}

// Error halts program execution with an error
func Error(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg starlark.Value
	if err := starlark.UnpackPositionalArgs("error", args, kwargs, 1, &msg); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("transform error: %s", msg)
}
