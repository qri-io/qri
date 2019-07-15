// Package startf implements dataset transformations using the starlark programming dialect
// For more info on starlark check github.com/google/starlark
package startf

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	skyctx "github.com/qri-io/qri/startf/context"
	skyds "github.com/qri-io/qri/startf/ds"
	skyqri "github.com/qri-io/qri/startf/qri"
	"github.com/qri-io/starlib"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

// ExecOpts defines options for execution
type ExecOpts struct {
	Node             *p2p.QriNode               // supply a QriNode to make the 'qri' module available in starlark
	AllowFloat       bool                       // allow floating-point numbers
	AllowSet         bool                       // allow set data type
	AllowLambda      bool                       // allow lambda expressions
	AllowNestedDef   bool                       // allow nested def statements
	Secrets          map[string]interface{}     // passed-in secrets (eg: API keys)
	Globals          starlark.StringDict        // global values to pass for script execution
	MutateFieldCheck func(path ...string) error // func that errors if field specified by path is mutated
	OutWriter        io.Writer                  // provide a writer to record script "stdout" to
	ModuleLoader     ModuleLoader               // starlark module loader function
}

// AddQriNodeOpt adds a qri node to execution options
func AddQriNodeOpt(node *p2p.QriNode) func(o *ExecOpts) {
	return func(o *ExecOpts) {
		o.Node = node
	}
}

// AddMutateFieldCheck provides a checkFunc to ExecScript
func AddMutateFieldCheck(check func(path ...string) error) func(o *ExecOpts) {
	return func(o *ExecOpts) {
		o.MutateFieldCheck = check
	}
}

// SetOutWriter provides a writer to record the "stderr" diagnostic output of the transform script
func SetOutWriter(w io.Writer) func(o *ExecOpts) {
	return func(o *ExecOpts) {
		if w != nil {
			o.OutWriter = w
		}
	}
}

// DefaultExecOpts applies default options to an ExecOpts pointer
func DefaultExecOpts(o *ExecOpts) {
	o.AllowFloat = true
	o.AllowSet = true
	o.AllowLambda = true
	o.Globals = starlark.StringDict{}
	o.OutWriter = ioutil.Discard
	o.ModuleLoader = DefaultModuleLoader
}

type transform struct {
	node         *p2p.QriNode
	next         *dataset.Dataset
	prev         *dataset.Dataset
	skyqri       *skyqri.Module
	checkFunc    func(path ...string) error
	globals      starlark.StringDict
	bodyFile     qfs.File
	stderr       io.Writer
	moduleLoader ModuleLoader

	download starlark.Iterable
}

// ModuleLoader is a function that can load starlark modules
type ModuleLoader func(thread *starlark.Thread, module string) (starlark.StringDict, error)

// DefaultModuleLoader is the loader ExecScript will use unless configured otherwise
var DefaultModuleLoader = func(thread *starlark.Thread, module string) (dict starlark.StringDict, err error) {
	return starlib.Loader(thread, module)
}

// ExecScript executes a transformation against a starlark script file. The next dataset pointer
// may be modified, while the prev dataset point is read-only. At a bare minimum this function
// will set transformation details, but starlark scripts can modify many parts of the dataset
// pointer, including meta, structure, and transform. opts may provide more ways for output to
// be produced from this function.
func ExecScript(next, prev *dataset.Dataset, opts ...func(o *ExecOpts)) error {
	var err error
	if next.Transform == nil || next.Transform.ScriptFile() == nil {
		return fmt.Errorf("no script to execute")
	}

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

	// set transform details
	next.Transform.Syntax = "starlark"
	next.Transform.SyntaxVersion = Version

	script := next.Transform.ScriptFile()
	// "tee" the script reader to avoid losing script data, as starlark.ExecFile
	// reads, data will be copied to buf, which is re-set to the transform script
	buf := &bytes.Buffer{}
	tr := io.TeeReader(script, buf)
	pipeScript := qfs.NewMemfileReader(script.FileName(), tr)

	t := &transform{
		node:         o.Node,
		next:         next,
		prev:         prev,
		skyqri:       skyqri.NewModule(o.Node),
		checkFunc:    o.MutateFieldCheck,
		stderr:       o.OutWriter,
		moduleLoader: o.ModuleLoader,
	}

	if o.Node != nil {
		// if node localstreams exists, write to both localstreams and output buffer
		t.stderr = io.MultiWriter(o.OutWriter, o.Node.LocalStreams.ErrOut)
	}

	ctx := skyctx.NewContext(next.Transform.Config, o.Secrets)

	thread := &starlark.Thread{
		Load: t.ModuleLoader,
		Print: func(thread *starlark.Thread, msg string) {
			// note we're ignoring a returned error here
			_, _ = t.stderr.Write([]byte(msg))
		},
	}

	// execute the transformation
	t.globals, err = starlark.ExecFile(thread, pipeScript.FileName(), pipeScript, t.locals())
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			return fmt.Errorf(evalErr.Backtrace())
		}
		return err
	}

	funcs, err := t.specialFuncs()
	if err != nil {
		return err
	}

	for name, fn := range funcs {
		val, err := fn(t, thread, ctx)

		if err != nil {
			if evalErr, ok := err.(*starlark.EvalError); ok {
				return fmt.Errorf(evalErr.Backtrace())
			}
			return err
		}

		ctx.SetResult(name, val)
	}

	err = callTransformFunc(t, thread, ctx)
	if evalErr, ok := err.(*starlark.EvalError); ok {
		return fmt.Errorf(evalErr.Backtrace())
	}

	// restore consumed script file
	next.Transform.SetScriptFile(qfs.NewMemfileBytes("transform.star", buf.Bytes()))

	return err
}

// Error halts program execution with an error
func Error(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg starlark.Value
	if err := starlark.UnpackPositionalArgs("error", args, kwargs, 1, &msg); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("transform error: %s", msg)
}

// ErrNotDefined is for when a starlark value is not defined or does not exist
var ErrNotDefined = fmt.Errorf("not defined")

// globalFunc checks if a global function is defined
func (t *transform) globalFunc(name string) (fn *starlark.Function, err error) {
	x, ok := t.globals[name]
	if !ok {
		return fn, ErrNotDefined
	}
	if x.Type() != "function" {
		return fn, fmt.Errorf("'%s' is not a function", name)
	}
	return x.(*starlark.Function), nil
}

func confirmIterable(x starlark.Value) (starlark.Iterable, error) {
	v, ok := x.(starlark.Iterable)
	if !ok {
		return nil, fmt.Errorf("did not return structured data")
	}
	return v, nil
}

func (t *transform) specialFuncs() (defined map[string]specialFunc, err error) {
	specialFuncs := map[string]specialFunc{
		"download": callDownloadFunc,
	}

	defined = map[string]specialFunc{}

	for name, fn := range specialFuncs {
		if _, err = t.globalFunc(name); err != nil {
			if err == ErrNotDefined {
				err = nil
				continue
			}
			return nil, err
		}
		defined[name] = fn
	}

	return
}

type specialFunc func(t *transform, thread *starlark.Thread, ctx *skyctx.Context) (result starlark.Value, err error)

func callDownloadFunc(t *transform, thread *starlark.Thread, ctx *skyctx.Context) (result starlark.Value, err error) {
	httpGuard.EnableNtwk()
	defer httpGuard.DisableNtwk()
	t.print("📡 running download...\n")

	var download *starlark.Function
	if download, err = t.globalFunc("download"); err != nil {
		if err == ErrNotDefined {
			return starlark.None, nil
		}
		return starlark.None, err
	}

	return starlark.Call(thread, download, starlark.Tuple{ctx.Struct()}, nil)
}

func callTransformFunc(t *transform, thread *starlark.Thread, ctx *skyctx.Context) (err error) {
	var transform *starlark.Function
	if transform, err = t.globalFunc("transform"); err != nil {
		if err == ErrNotDefined {
			return nil
		}
		return err
	}
	t.print("🤖  running transform...\n")

	d := skyds.NewDataset(t.prev, t.checkFunc)
	d.SetMutable(t.next)
	if _, err = starlark.Call(thread, transform, starlark.Tuple{d.Methods(), ctx.Struct()}, nil); err != nil {
		return err
	}
	return nil
}

func (t *transform) setSpinnerMsg(msg string) {
	if t.node != nil {
		t.node.LocalStreams.SpinnerMsg(msg)
	}
}

// print writes output only if a node is specified
func (t *transform) print(msg string) {
	t.stderr.Write([]byte(msg))
}

func (t *transform) locals() starlark.StringDict {
	return starlark.StringDict{
		"load_dataset": starlark.NewBuiltin("load_dataset", t.LoadDataset),
	}
}

// ModuleLoader sums all loading assets to resolve a module name during transform execution
func (t *transform) ModuleLoader(thread *starlark.Thread, module string) (dict starlark.StringDict, err error) {
	if module == skyqri.ModuleName && t.skyqri != nil {
		return t.skyqri.Namespace(), nil
	}

	if t.moduleLoader == nil {
		return nil, fmt.Errorf("couldn't load module: %s", module)
	}

	return t.moduleLoader(thread, module)
}

// LoadDataset is a function
func (t *transform) LoadDataset(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var refstr starlark.String
	if err := starlark.UnpackArgs("load_dataset", args, kwargs, "ref", &refstr); err != nil {
		return starlark.None, err
	}

	ds, err := t.loadDataset(refstr.GoString())
	if err != nil {
		return starlark.None, err
	}

	return skyds.NewDataset(ds, nil).Methods(), nil
}

func (t *transform) loadDataset(refstr string) (*dataset.Dataset, error) {
	if t.node == nil {
		return nil, fmt.Errorf("no qri node available to load dataset: %s", refstr)
	}

	ref, err := repo.ParseDatasetRef(refstr)
	if err != nil {
		return nil, err
	}
	if err := repo.CanonicalizeDatasetRef(t.node.Repo, &ref); err != nil {
		return nil, err
	}
	t.node.LocalStreams.PrintErr(fmt.Sprintf("load: %s\n", ref.String()))

	ds, err := dsfs.LoadDataset(t.node.Repo.Store(), ref.Path)
	if err != nil {
		return nil, err
	}

	if ds.BodyFile() == nil {
		if err = ds.OpenBodyFile(t.node.Repo.Filesystem()); err != nil {
			return nil, err
		}
	}

	if t.next.Transform.Resources == nil {
		t.next.Transform.Resources = map[string]*dataset.TransformResource{}
	}
	t.next.Transform.Resources[ref.Path] = &dataset.TransformResource{Path: ref.String()}

	return ds, nil
}
