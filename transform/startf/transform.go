package startf

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/preview"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
	stards "github.com/qri-io/qri/transform/startf/ds"
	"github.com/qri-io/qri/version"
	"github.com/qri-io/starlib"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
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
	// filesystem for loading scripts
	Filesystem qfs.Filesystem
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

// AddFilesystem adds a filesystem to the transformer
func AddFilesystem(fs qfs.Filesystem) func(o *ExecOpts) {
	return func(o *ExecOpts) {
		o.Filesystem = fs
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

// SetErrWriter provides a writer to record the "stderr" diagnostic output of
// the transform script
func SetErrWriter(w io.Writer) func(o *ExecOpts) {
	return func(o *ExecOpts) {
		o.ErrWriter = w
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
	config       map[string]interface{}
	secrets      map[string]interface{}
	fs           qfs.Filesystem
	dsLoader     dsref.Loader
	stards       *stards.BoundDataset
	globals      starlark.StringDict
	eventsCh     chan event.Event
	writer       io.Writer
	thread       *starlark.Thread
	changeSet    map[string]struct{}
	commitCalled bool
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
	resolve.AllowGlobalReassign = true

	// add error func to starlark environment
	starlark.Universe["error"] = starlark.NewBuiltin("error", Error)
	for key, val := range o.Globals {
		starlark.Universe[key] = val
	}

	thread := &starlark.Thread{
		Load: o.ModuleLoader,
		Print: func(thread *starlark.Thread, msg string) {
			if o.EventsCh != nil {
				o.EventsCh <- event.Event{
					Type: event.ETTransformPrint,
					Payload: event.TransformMessage{
						Msg: msg,
					},
				}
			}
			o.ErrWriter.Write([]byte(msg + "\n"))
		},
	}

	r := &StepRunner{
		config:    target.Transform.Config,
		secrets:   o.Secrets,
		fs:        o.Filesystem,
		dsLoader:  o.DatasetLoader,
		eventsCh:  o.EventsCh,
		writer:    o.ErrWriter,
		thread:    thread,
		globals:   starlark.StringDict{},
		changeSet: o.ChangeSet,
	}
	r.stards = stards.NewBoundDataset(target, r.onCommit)

	return r
}

// RunStep runs the single transform step using the dataset
func (r *StepRunner) RunStep(ctx context.Context, ds *dataset.Dataset, st *dataset.TransformStep) (err error) {
	r.globals["load_dataset"] = starlark.NewBuiltin("load_dataset", r.loadDatasetFunc(ctx, ds))
	r.globals["dataset"] = r.stards
	r.globals["config"] = config(r.config)
	r.globals["secrets"] = secrets(r.secrets)

	script, ok := st.Script.(string)
	if !ok {
		return fmt.Errorf("starlark step Script must be a string. got %T", st.Script)
	}

	// Recover from errors.
	defer func() {
		if r := recover(); r != nil {
			// Need to assign to the named return value from
			// a recovery
			err = fmt.Errorf("running transform: %w", r)
		}
	}()

	// Parse, resolve, and compile a Starlark source file.
	file, mod, err := starlark.SourceProgram(fmt.Sprintf("%s.star", st.Name), strings.NewReader(script), r.globals.Has)
	if err != nil {
		return err
	}

	r.printFinalStatement(file)

	globals, err := mod.Init(r.thread, r.globals)
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			return fmt.Errorf(evalErr.Backtrace())
		}
		return err
	}
	for key, val := range globals {
		r.globals[key] = val
	}

	return
}

// TODO(b5): this needs to be finished
func (r *StepRunner) printFinalStatement(f *syntax.File) {
	if len(f.Stmts) == 0 {
		return
	}

	_, stepEnd := f.Span()
	lastStmt := f.Stmts[len(f.Stmts)-1]
	_, end := lastStmt.Span()

	// only print if statment is on the last line
	if end.Line == stepEnd.Line {
		// r.eventsCh <- event.Event{
		// 	Type: event.ETTransformPrint,
		// 	Payload: event.TransformMessage{
		// 		Msg: fmt.Sprintf("%T %#v\n", lastStmt, lastStmt),
		// 	},
		// }
	}
}

// CommitCalled returns true if the commit function has been called
func (r *StepRunner) CommitCalled() bool {
	return r.commitCalled
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

// loadDatasetFunc returns an implementation of the starlark load_dataset
// function
func (r *StepRunner) loadDatasetFunc(ctx context.Context, target *dataset.Dataset) func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var refstr starlark.String
		if err := starlark.UnpackArgs("load_dataset", args, kwargs, "ref", &refstr); err != nil {
			return starlark.None, err
		}

		if r.dsLoader == nil {
			return nil, fmt.Errorf("load_datset function is not enabled")
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

// func (r *StepRunner) print(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
// 	var (
// 		str string
// 		message starlark.Value
// 	)

// 	if err := starlark.UnpackArgs("print", args, kwargs, "message", &message); err != nil {
// 		return starlark.None, err
// 	}

// 	if stringer, ok := message.(starlark.GoString)
// 	if r.eventsCh != nil {
// 		r.eventsCh <- event.Event{
// 			Type: event.ETTransformPrint,
// 			Payload: event.TransformMessage{
// 				Msg: message.GoString(),
// 			},
// 		}
// 	}
// 	r.writer.Write([]byte(message.GoString() + "\n"))
// 	return starlark.None, nil
// }

func (r *StepRunner) onCommit(ds *stards.Dataset) error {
	// Which components were changed
	if r.changeSet != nil {
		changes := ds.Changes()
		for comp := range changes {
			r.changeSet[comp] = changes[comp]
		}
	}

	ctx := context.TODO()
	if err := ds.AssignComponentsFromDataframe(ctx, r.changeSet, r.fs, r.dsLoader); err != nil {
		return err
	}

	if r.eventsCh != nil {
		pview, err := preview.Create(context.TODO(), ds.Dataset())
		if err != nil {
			return err
		}
		r.eventsCh <- event.Event{Type: event.ETTransformDatasetPreview, Payload: pview}
	}
	r.commitCalled = true
	return nil
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
