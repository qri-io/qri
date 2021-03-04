package lib

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/qri-io/qri/auth/token"
	"github.com/qri-io/qri/profile"
)

// dispatcher isolates the dispatch method
type dispatcher interface {
	Dispatch(ctx context.Context, method string, param interface{}) (res interface{}, err error)
}

// Dispatch is a system for handling calls to lib. Should only be called by top-level lib methods.
//
// When programs are using qri as a library (such as the `cmd` package), calls to `lib` will
// arrive at dispatch, before being routed to the actual implementation routine. This solves
// a few problems:
//   1) Multiple methods can be running on qri at once, dispatch will schedule as needed (TODO)
//   2) Access to core qri data structures (like logbook) can be handled safetly (TODO)
//   3) User identity, permissions, etc is scoped to a single call, not the global process (TODO)
//   4) The qri http api maps directly onto dispatch's behavior, leading to a simpler api
//   5) A `qri connect` process can be transparently forwarded a method call with little work
//
// At construction time, the Instance registers all methods that dispatch can access, as well
// as the input and output parameters for those methods, and associates a string name for each
// method. Dispatch works by looking up that method name, constructing the necessary input,
// then invoking the actual implementation.
func (inst *Instance) Dispatch(ctx context.Context, method string, param interface{}) (res interface{}, err error) {
	if inst == nil {
		return nil, fmt.Errorf("instance is nil, cannot dispatch")
	}

	// If the http rpc layer is engaged, use it to dispatch methods
	// This happens when another process is running `qri connect`
	if inst.http != nil {
		if tok := token.FromCtx(ctx); tok == "" {
			// If no token exists, create one from configured profile private key &
			// add it to the request context
			// TODO(b5): we're falling back to the configured user to make requests,
			// is this the right default?
			p, err := profile.NewProfile(inst.cfg.Profile)
			if err != nil {
				return nil, err
			}
			tokstr, err := token.NewPrivKeyAuthToken(p.PrivKey, time.Minute)
			if err != nil {
				return nil, err
			}
			ctx = token.AddToContext(ctx, tokstr)
		}

		if c, ok := inst.regMethods.lookup(method); ok {
			// TODO(dustmop): This is always using the "POST" verb currently. We need some
			// mechanism of tagging methods as being read-only and "GET"-able. Once that
			// exists, use it here to lookup the verb that should be used to invoke the rpc.
			out := reflect.New(c.OutType)
			res = out.Interface()
			err = inst.http.Call(ctx, methodEndpoint(method), param, res)
			if err != nil {
				return nil, err
			}
			out = reflect.ValueOf(res)
			out = out.Elem()
			return out.Interface(), nil
		}
		return nil, fmt.Errorf("method %q not found", method)
	}

	// Look up the method for the given signifier
	if c, ok := inst.regMethods.lookup(method); ok {
		// Construct the isolated scope for this call
		// TODO(dustmop): Add user authentication, profile, identity, etc
		// TODO(dustmop): Also determine if the method is read-only vs read-write,
		// and only execute a single read-write method at a time
		// Eventually, the data that lives in scope should be immutable for its lifetime,
		// or use copy-on-write semantics, so that one method running at the same time as
		// another cannot modify the out-of-scope data of the other. This will mostly
		// involve making copies of the right things
		scope, err := newScope(ctx, inst)
		if err != nil {
			return nil, err
		}

		// Construct the parameter list for the function call, then call it
		args := make([]reflect.Value, 3)
		args[0] = reflect.ValueOf(c.Impl)
		args[1] = reflect.ValueOf(scope)
		args[2] = reflect.ValueOf(param)
		outVals := c.Func.Call(args)

		// TODO(dustmop): If the method wrote to our internal data structures, like
		// refstore, logbook, etc, serialize and commit those changes here

		// Validate the return values. This shouldn't fail as long as all method
		// implementations are declared correctly
		if len(outVals) != 2 {
			return nil, fmt.Errorf("wrong number of return args: %d", len(outVals))
		}

		// Extract the concrete typed values from the method return
		var out interface{}
		out = outVals[0].Interface()
		errVal := outVals[1].Interface()
		if errVal == nil {
			return out, nil
		}
		if err, ok := errVal.(error); ok {
			return out, err
		}
		return nil, fmt.Errorf("second return value should be an error, got: %v", errVal)
	}
	return nil, fmt.Errorf("method %q not found", method)
}

// NewInputParam takes a method name that has been registered, and constructs
// an instance of that input parameter
func (inst *Instance) NewInputParam(method string) interface{} {
	if c, ok := inst.regMethods.lookup(method); ok {
		obj := reflect.New(c.InType)
		return obj.Interface()
	}
	return nil
}

// regMethodSet represents a set of registered methods
type regMethodSet struct {
	reg map[string]callable
}

// lookup finds the callable structure with the given method name
func (r *regMethodSet) lookup(method string) (*callable, bool) {
	if c, ok := r.reg[method]; ok {
		return &c, true
	}
	return nil, false
}

type callable struct {
	Impl    interface{}
	Func    reflect.Value
	InType  reflect.Type
	OutType reflect.Type
}

// RegisterMethods iterates the methods provided by the lib API, and makes them visible to dispatch
func (inst *Instance) RegisterMethods() {
	reg := make(map[string]callable)
	// TODO(dustmop): Change registerOne to take both the MethodSet and the Impl, validate
	// that their signatures agree.
	inst.registerOne("fsi", &FSIImpl{}, reg)
	inst.registerOne("access", accessImpl{}, reg)
	inst.regMethods = &regMethodSet{reg: reg}
}

func (inst *Instance) registerOne(ourName string, impl interface{}, reg map[string]callable) {
	implType := reflect.TypeOf(impl)
	// Iterate methods on the implementation, register those that have the right signature
	num := implType.NumMethod()
	for k := 0; k < num; k++ {
		m := implType.Method(k)
		lowerName := strings.ToLower(m.Name)
		funcName := fmt.Sprintf("%s.%s", ourName, lowerName)

		// Validate the parameters to the method
		// should have 3 input parameters: (receiver, scope, input struct)
		// should have 2 output parametres: (output value, error)
		// TODO(dustmop): allow variadic returns: error only, cursor for pagination
		f := m.Type
		if f.NumIn() != 3 {
			log.Fatalf("%s: bad number of inputs: %d", funcName, f.NumIn())
		}
		if f.NumOut() != 2 {
			log.Fatalf("%s: bad number of outputs: %d", funcName, f.NumOut())
		}
		// First input must be the receiver
		inType := f.In(0)
		if inType != implType {
			log.Fatalf("%s: first input param should be impl, got %v", funcName, inType)
		}
		// Second input must be a scope
		inType = f.In(1)
		if inType.Name() != "scope" {
			log.Fatalf("%s: second input param should be scope, got %v", funcName, inType)
		}
		// Third input is a pointer to the input struct
		inType = f.In(2)
		if inType.Kind() != reflect.Ptr {
			log.Fatalf("%s: third input param must be a struct pointer, got %v", funcName, inType)
		}
		inType = inType.Elem()
		if inType.Kind() != reflect.Struct {
			log.Fatalf("%s: third input param must be a struct pointer, got %v", funcName, inType)
		}
		// First output is anything
		outType := f.Out(0)
		// Second output must be an error
		outErrType := f.Out(1)
		if outErrType.Name() != "error" {
			log.Fatalf("%s: second output param should be error, got %v", funcName, outErrType)
		}

		// Save the method to the registration table
		reg[funcName] = callable{
			Impl:    impl,
			Func:    m.Func,
			InType:  inType,
			OutType: outType,
		}
		log.Debugf("%d: registered %s(*%s) %v", k, funcName, inType, outType)
	}
}

// MethodSet represents a set of methods to be registered
type MethodSet interface {
	Name() string
}

func dispatchMethodName(m MethodSet, funcName string) string {
	lowerName := strings.ToLower(funcName)
	return fmt.Sprintf("%s.%s", m.Name(), lowerName)
}

// methodEndpoint returns a method name and returns the API endpoint for it
func methodEndpoint(method string) APIEndpoint {
	// TODO(dustmop): This is here temporarily. /fsi/write/ works differently than
	// other methods; their http API endpoints are only their method name, for
	// exmaple /status/. This should be replaced with an explicit mapping from
	// method names to endpoints.
	if method == "fsi.write" {
		return "/fsi/write/"
	}
	if method == "fsi.createlink" {
		return "/fsi/createlink/"
	}
	if method == "fsi.unlink" {
		return "/fsi/unlink/"
	}
	pos := strings.Index(method, ".")
	prefix := method[:pos]
	_ = prefix
	res := "/" + method[pos+1:] + "/"
	return APIEndpoint(res)
}

func dispatchReturnError(got interface{}, err error) error {
	if got != nil {
		log.Errorf("type mismatch: %v of type %s", got, reflect.TypeOf(got))
	}
	return err
}
