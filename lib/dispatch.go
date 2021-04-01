package lib

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/qri-io/qri/auth/token"
	"github.com/qri-io/qri/profile"
)

var (
	// ErrDispatchNilInstance indicates that the instance that dispatch as been called on is nil
	ErrDispatchNilInstance = errors.New("instance is nil, cannot dispatch")
	// ErrDispatchNilParam indicates that the param passed to dispatch is nil
	ErrDispatchNilParam = errors.New("param is nil, cannot dispatch")
)

// dispatcher isolates the dispatch method
type dispatcher interface {
	Dispatch(ctx context.Context, method string, param interface{}) (interface{}, Cursor, error)
}

// Cursor is used to paginate results for methods that support it
type Cursor interface{}

// MethodSet represents a set of methods to be registered
// Each registered method should have 2 input parameters and 1-3 output values
//   Input: (context.Context, input struct)
//   Output, 1: (error)
//           2: (output, error)
//           3: (output, Cursor, error)
// The implementation should have the same input and output as the method, except
// with the context.Context replaced by a scope.
// No other functions are allowed to be defined, other that those that are going to
// be registered (as described above) and those that are required by the interface.
type MethodSet interface {
	Name() string
	Attributes() map[string]AttributeSet
}

// AttributeSet is extra information about each method, such as: http endpoint,
// http verb, (TODO) permissions, and (TODO) other metadata
// Each method is required to have associated attributes in order to successfully register
type AttributeSet struct {
	endpoint APIEndpoint
	verb     string
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
// then invoking the actual implementation. Dispatch returns the custom value from the
// implementation, then a non-nil Cursor if the method supports pagination, then an error or nil.
func (inst *Instance) Dispatch(ctx context.Context, method string, param interface{}) (res interface{}, cur Cursor, err error) {
	source := ""
	return inst.dispatchMethodCall(ctx, method, param, source)
}

// Dispatch calls the same instance Dispatch but with an explicit source for ref resolution
func (isw *InstanceSourceWrap) Dispatch(ctx context.Context, method string, param interface{}) (res interface{}, cur Cursor, err error) {
	return isw.inst.dispatchMethodCall(ctx, method, param, isw.source)
}

func (inst *Instance) dispatchMethodCall(ctx context.Context, method string, param interface{}, source string) (res interface{}, cur Cursor, err error) {
	if inst == nil {
		return nil, nil, ErrDispatchNilInstance
	}
	if param == nil || (reflect.ValueOf(param).Kind() == reflect.Ptr && reflect.ValueOf(param).IsNil()) {
		return nil, nil, ErrDispatchNilParam
	}

	// If the input parameters has a Validate method, call it
	if validator, ok := param.(ParamValidator); ok {
		err = validator.Validate()
		if err != nil {
			return nil, nil, err
		}
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
				return nil, nil, err
			}
			tokstr, err := token.NewPrivKeyAuthToken(p.PrivKey, p.ID.String(), time.Minute)
			if err != nil {
				return nil, nil, err
			}
			ctx = token.AddToContext(ctx, tokstr)
		}

		if c, ok := inst.regMethods.lookup(method); ok {
			if c.Endpoint == "" {
				return nil, nil, ErrUnsupportedRPC
			}
			if c.OutType != nil {
				out := reflect.New(c.OutType)
				res = out.Interface()
			}
			// TODO(dustmop): Send the source across the RPC, using an HTTP header
			// TODO(ramfox): dispatch is still unable to give enough details to the url
			// (because it doesn't know how or what param information to put into the url or query)
			// for it to reliably use GET. All POSTs w/ content type application json work, however.
			// we may want to just flat out say that as an RPC layer, dispatch will only ever use
			// json POST to communicate.
			err = inst.http.CallMethod(ctx, c.Endpoint, "POST", param, res)
			if err != nil {
				return nil, nil, err
			}
			cur = nil
			var inf interface{}
			if res != nil {
				out := reflect.ValueOf(res)
				out = out.Elem()
				inf = out.Interface()
			}
			return inf, cur, nil
		}
		return nil, nil, fmt.Errorf("method %q not found", method)
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
		scope, err := newScope(ctx, inst, source)
		if err != nil {
			return nil, nil, err
		}

		// Handle filepaths in the params by calling qfs.Abs on each of them
		param = normalizeInputParams(param)

		// Construct the parameter list for the function call, then call it
		args := make([]reflect.Value, 3)
		args[0] = reflect.ValueOf(c.Impl)
		args[1] = reflect.ValueOf(scope)
		args[2] = reflect.ValueOf(param)
		outVals := c.Func.Call(args)

		// TODO(dustmop): If the method wrote to our internal data structures, like
		// refstore, logbook, etc, serialize and commit those changes here

		// Validate the return values.
		if len(outVals) < 1 || len(outVals) > 3 {
			return nil, nil, fmt.Errorf("wrong number of return values: %d", len(outVals))
		}
		// Extract the concrete typed values from the method return
		var out interface{}
		var cur Cursor
		// There are either 1, 2, or 3 output values:
		//   1: func() (err)
		//   2: func() (out, err)
		//   3: func() (out, cur, err)
		if len(outVals) == 2 || len(outVals) == 3 {
			out = outVals[0].Interface()
		}
		if len(outVals) == 3 {
			cur = outVals[1].Interface()
		}
		// Error always comes last
		errVal := outVals[len(outVals)-1].Interface()
		if errVal == nil {
			return out, cur, nil
		}
		if err, ok := errVal.(error); ok {
			return out, cur, err
		}
		return nil, nil, fmt.Errorf("last return value should be an error, got: %v", errVal)
	}
	return nil, nil, fmt.Errorf("method %q not found", method)
}

// ParamValidator may be implemented by method parameter structs, and if so
// then Dispatch will validate the parameters are okay before calling anything
type ParamValidator interface {
	Validate() error
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
	Impl      interface{}
	Func      reflect.Value
	InType    reflect.Type
	OutType   reflect.Type
	RetCursor bool
	Endpoint  APIEndpoint
	Verb      string
}

// RegisterMethods iterates the methods provided by the lib API, and makes them visible to dispatch
func (inst *Instance) RegisterMethods() {
	reg := make(map[string]callable)
	inst.registerOne("access", inst.Access(), accessImpl{}, reg)
	inst.registerOne("collection", inst.Collection(), collectionImpl{}, reg)
	inst.registerOne("config", inst.Config(), configImpl{}, reg)
	inst.registerOne("dataset", inst.Dataset(), datasetImpl{}, reg)
	inst.registerOne("diff", inst.Diff(), diffImpl{}, reg)
	inst.registerOne("fsi", inst.Filesys(), fsiImpl{}, reg)
	inst.registerOne("log", inst.Log(), logImpl{}, reg)
	inst.registerOne("peer", inst.Peer(), peerImpl{}, reg)
	inst.registerOne("profile", inst.Profile(), profileImpl{}, reg)
	inst.registerOne("registry", inst.Registry(), registryImpl{}, reg)
	inst.registerOne("remote", inst.Remote(), remoteImpl{}, reg)
	inst.registerOne("search", inst.Search(), searchImpl{}, reg)
	inst.registerOne("sql", inst.SQL(), sqlImpl{}, reg)
	inst.registerOne("automation", inst.Automation(), automationImpl{}, reg)
	inst.regMethods = &regMethodSet{reg: reg}
}

func (inst *Instance) registerOne(ourName string, methods MethodSet, impl interface{}, reg map[string]callable) {
	implType := reflect.TypeOf(impl)
	msetType := reflect.TypeOf(methods)
	methodMap := inst.buildMethodMap(methods)
	// Validate that the methodSet has the correct name
	if methods.Name() != ourName {
		regFail("registration wrong name, expect: %q, got: %q", ourName, methods.Name())
	}
	// Iterate methods on the implementation, register those that have the right signature
	num := implType.NumMethod()
	for k := 0; k < num; k++ {
		i := implType.Method(k)
		lowerName := strings.ToLower(i.Name)
		funcName := fmt.Sprintf("%s.%s", ourName, lowerName)

		// Validate the parameters to the implementation
		// should have 3 input parameters: (receiver, scope, input struct)
		// should have 1-3 output parametres: ([output value]?, [cursor]?, error)
		f := i.Type
		if f.NumIn() != 3 {
			regFail("%s: bad number of inputs: %d", funcName, f.NumIn())
		}
		// First input must be the receiver
		inType := f.In(0)
		if inType != implType {
			regFail("%s: first input param should be impl, got %v", funcName, inType)
		}
		// Second input must be a scope
		inType = f.In(1)
		if inType.Name() != "scope" {
			regFail("%s: second input param should be scope, got %v", funcName, inType)
		}
		// Third input is a pointer to the input struct
		inType = f.In(2)
		if inType.Kind() != reflect.Ptr {
			regFail("%s: third input param must be a struct pointer, got %v", funcName, inType)
		}
		inType = inType.Elem()
		if inType.Kind() != reflect.Struct {
			regFail("%s: third input param must be a struct pointer, got %v", funcName, inType)
		}
		// Validate the output values of the implementation
		numOuts := f.NumOut()
		if numOuts < 1 || numOuts > 3 {
			regFail("%s: bad number of outputs: %d", funcName, numOuts)
		}
		// Validate output values
		var outType reflect.Type
		returnsCursor := false
		if numOuts == 2 || numOuts == 3 {
			// First output is anything
			outType = f.Out(0)
		}
		if numOuts == 3 {
			// Second output must be a cursor
			outCursorType := f.Out(1)
			if outCursorType.Name() != "Cursor" {
				regFail("%s: second output val must be a cursor, got %v", funcName, outCursorType)
			}
			returnsCursor = true
		}
		// Last output must be an error
		outErrType := f.Out(numOuts - 1)
		if outErrType.Name() != "error" {
			regFail("%s: last output val should be error, got %v", funcName, outErrType)
		}

		// Validate the parameters to the method that matches the implementation
		// should have 3 input parameters: (receiver, context.Context, input struct: same as impl])
		// should have 1-3 output parametres: ([output value: same as impl], [cursor], error)
		m, ok := methodMap[i.Name]
		if !ok {
			regFail("method %s not found on MethodSet", i.Name)
		}
		f = m.Type
		if f.NumIn() != 3 {
			regFail("%s: bad number of inputs: %d", funcName, f.NumIn())
		}
		// First input must be the receiver
		mType := f.In(0)
		if mType.Name() != msetType.Name() {
			regFail("%s: first input param should be impl, got %v", funcName, mType)
		}
		// Second input must be a context
		mType = f.In(1)
		if mType.Name() != "Context" {
			regFail("%s: second input param should be context.Context, got %v", funcName, mType)
		}
		// Third input is a pointer to the input struct
		mType = f.In(2)
		if mType.Kind() != reflect.Ptr {
			regFail("%s: third input param must be a pointer, got %v", funcName, mType)
		}
		mType = mType.Elem()
		if mType != inType {
			regFail("%s: third input param must match impl, expect %v, got %v", funcName, inType, mType)
		}
		// Validate the output values of the implementation
		msetNumOuts := f.NumOut()
		if msetNumOuts < 1 || msetNumOuts > 3 {
			regFail("%s: bad number of outputs: %d", funcName, f.NumOut())
		}
		// First output, if there's more than 1, matches the impl output
		if msetNumOuts == 2 || msetNumOuts == 3 {
			mType = f.Out(0)
			if mType != outType {
				regFail("%s: first output val must match impl, expect %v, got %v", funcName, outType, mType)
			}
		}
		// Second output, if there are three, must be a cursor
		if msetNumOuts == 3 {
			mType = f.Out(1)
			if mType.Name() != "Cursor" {
				regFail("%s: second output val must match a cursor, got %v", funcName, mType)
			}
		}
		// Last output must be an error
		mType = f.Out(msetNumOuts - 1)
		if mType.Name() != "error" {
			regFail("%s: last output val should be error, got %v", funcName, mType)
		}

		// Remove this method from the methodSetMap now that it has been processed
		delete(methodMap, i.Name)

		var endpoint APIEndpoint
		var httpVerb string
		// Additional attributes for the method are found in the Attributes
		amap := methods.Attributes()
		methodAttrs, ok := amap[lowerName]
		if !ok {
			regFail("not in Attributes: %s.%s", ourName, lowerName)
		}
		endpoint = methodAttrs.endpoint
		httpVerb = methodAttrs.verb
		// If both these are empty string, RPC is not allowed for this method
		if endpoint != "" || httpVerb != "" {
			if !strings.HasPrefix(string(endpoint), "/") {
				regFail("%s: endpoint URL must start with /, got %q", lowerName, endpoint)
			}
			if httpVerb != http.MethodGet && httpVerb != http.MethodPost && httpVerb != http.MethodPut {
				regFail("%s: unknown http verb, got %q", lowerName, httpVerb)
			}
		}

		// Save the method to the registration table
		reg[funcName] = callable{
			Impl:      impl,
			Func:      i.Func,
			InType:    inType,
			OutType:   outType,
			RetCursor: returnsCursor,
			Endpoint:  endpoint,
			Verb:      httpVerb,
		}
		log.Debugf("%d: registered %s(*%s) %v", k, funcName, inType, outType)
	}

	for k := range methodMap {
		if k != "Name" && k != "Attributes" {
			regFail("%s: did not find implementation for method %s", msetType, k)
		}
	}
}

func regFail(fstr string, vals ...interface{}) {
	panic(fmt.Sprintf(fstr, vals...))
}

func (inst *Instance) buildMethodMap(impl interface{}) map[string]reflect.Method {
	result := make(map[string]reflect.Method)
	implType := reflect.TypeOf(impl)
	num := implType.NumMethod()
	for k := 0; k < num; k++ {
		m := implType.Method(k)
		result[m.Name] = m
	}
	return result
}

func dispatchMethodName(m MethodSet, funcName string) string {
	lowerName := strings.ToLower(funcName)
	return fmt.Sprintf("%s.%s", m.Name(), lowerName)
}

func dispatchReturnError(got interface{}, err error) error {
	if got != nil {
		log.Errorf("type mismatch: %v of type %s", got, reflect.TypeOf(got))
	}
	return err
}
