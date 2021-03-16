package lib

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qri-io/qri/api/util"
)

func TestRegisterMethods(t *testing.T) {
	ctx := context.Background()

	inst, cleanup := NewMemTestInstance(ctx, t)
	defer cleanup()
	m := &animalMethods{d: inst}

	reg := make(map[string]callable)
	inst.registerOne("animal", m, animalImpl{}, reg)

	expectToPanic(t, func() {
		reg = make(map[string]callable)
		inst.registerOne("animal", m, badAnimalOneImpl{}, reg)
	}, "animal.cat: bad number of outputs: 0")

	expectToPanic(t, func() {
		reg = make(map[string]callable)
		inst.registerOne("animal", m, badAnimalTwoImpl{}, reg)
	}, "method Doggie not found on MethodSet")

	expectToPanic(t, func() {
		reg = make(map[string]callable)
		inst.registerOne("animal", m, badAnimalThreeImpl{}, reg)
	}, "animal.cat: second input param should be scope, got context.Context")

	expectToPanic(t, func() {
		reg = make(map[string]callable)
		inst.registerOne("animal", m, badAnimalFourImpl{}, reg)
	}, "animal.dog: third input param must be a struct pointer, got string")

	expectToPanic(t, func() {
		reg = make(map[string]callable)
		inst.registerOne("animal", m, badAnimalFiveImpl{}, reg)
	}, "*lib.animalMethods: did not find implementation for method Dog")
}

func TestRegisterVariadicReturn(t *testing.T) {
	ctx := context.Background()

	inst, cleanup := NewMemTestInstance(ctx, t)
	defer cleanup()
	f := &fruitMethods{d: inst}

	reg := make(map[string]callable)
	inst.registerOne("fruit", f, fruitImpl{}, reg)
	inst.regMethods = &regMethodSet{reg: reg}

	_, _, err := inst.Dispatch(ctx, "fruit.apple", &fruitParams{})
	expectErr := "no more apples"
	if expectErr != err.Error() {
		t.Errorf("apple return mismtach, expect: %q, got: %q", expectErr, err)
	}

	got, cur, err := inst.Dispatch(ctx, "fruit.banana", &fruitParams{})
	if got != "batman" {
		t.Errorf("banana return mismatch, expect: batman, got: %q", got)
	}
	if cur != nil {
		t.Errorf("banana return mismatch, expect: nil cursor, got: %q", cur)
	}
	if err.Error() != "success" {
		t.Errorf("banana return mismatch, expect: success error, got: %q", err)
	}
}

func TestVariadicReturnsWorkOverHTTP(t *testing.T) {
	ctx := context.Background()

	// Instance that registers the fruit methods
	servInst, servCleanup := NewMemTestInstance(ctx, t)
	defer servCleanup()
	servFruit := &fruitMethods{d: servInst}
	reg := make(map[string]callable)
	servInst.registerOne("fruit", servFruit, fruitImpl{}, reg)
	servInst.regMethods = &regMethodSet{reg: reg}

	// A local call, no RPC used
	err := servFruit.Apple(ctx, &fruitParams{})
	expectErr := "no more apples"
	if err.Error() != expectErr {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err)
	}

	// Instance that acts as a client of another
	clientInst, clientCleanup := NewMemTestInstance(ctx, t)
	defer clientCleanup()
	clientFruit := &fruitMethods{d: clientInst}
	reg = make(map[string]callable)
	clientInst.registerOne("fruit", clientFruit, fruitImpl{}, reg)
	clientInst.regMethods = &regMethodSet{reg: reg}

	// Run the first instance in "connect" mode, tell the second
	// instance to use it for RPC calls
	httpClient, connectCleanup := serverConnectAndListen(t, servInst, 7890)
	defer connectCleanup()
	clientInst.http = httpClient

	// Call the method, which will be send over RPC
	err = clientFruit.Apple(ctx, &fruitParams{})
	if err == nil {
		t.Fatal("expected to get error but did not get one")
	}
	expectErr = newHTTPResponseError("no more apples")
	if err.Error() != expectErr {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err)
	}

	// Call another method
	_, _, err = clientFruit.Banana(ctx, &fruitParams{})
	if err == nil {
		t.Fatal("expected to get error but did not get one")
	}
	expectErr = newHTTPResponseError("success")
	if err.Error() != expectErr {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err)
	}

	// Call another method, which won't return an error
	err = clientFruit.Cherry(ctx, &fruitParams{})
	if err != nil {
		t.Errorf("%s", err)
	}

	// Call a method successfully
	val, _, err := clientFruit.Date(ctx, &fruitParams{})
	if err != nil {
		t.Errorf("%s", err)
	}
	if val != "January 1st" {
		t.Errorf("value mismatch, expect: January 1st, got: %s", val)
	}

	// Call a method not supported over RPC
	val, _, err = clientFruit.Entawak(ctx, &fruitParams{})
	if err == nil {
		t.Fatal("expected to get error but did not get one")
	}
	expectErr = "method is not suported over RPC"
	if err.Error() != expectErr {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err)
	}
}

func serverConnectAndListen(t *testing.T, servInst *Instance, port int) (*HTTPClient, func()) {
	address := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)
	connection, err := NewHTTPClient(address)
	if err != nil {
		t.Fatal(err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		method := ""
		if r.URL.Path == "/apple" {
			method = "fruit.apple"
		} else if r.URL.Path == "/banana" {
			method = "fruit.banana"
		} else if r.URL.Path == "/cherry" {
			method = "fruit.cherry"
		} else if r.URL.Path == "/date" {
			method = "fruit.date"
		} else if r.URL.Path == "/entawak" {
			method = "fruit.entawak"
		} else {
			t.Fatalf("404: Not Found %q", r.URL.Path)
		}
		p := servInst.NewInputParam(method)
		res, _, err := servInst.Dispatch(r.Context(), method, p)
		if err != nil {
			util.RespondWithError(w, err)
			return
		}
		util.WriteResponse(w, res)
	}
	mockAPIServer := httptest.NewUnstartedServer(http.HandlerFunc(handler))
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatal(err.Error())
	}
	mockAPIServer.Listener = listener
	mockAPIServer.Start()
	apiServerCleanup := func() {
		mockAPIServer.Close()
	}
	return connection, apiServerCleanup
}

func newHTTPResponseError(msg string) string {
	tmpl := `{
  "meta": {
    "code": 500,
    "error": "%s"
  }
}`
	return fmt.Sprintf(tmpl, msg)
}

func expectToPanic(t *testing.T, regFunc func(), expectMessage string) {
	t.Helper()

	doneCh := make(chan error)
	panicMessage := ""

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicMessage = fmt.Sprintf("%s", r)
			}
			doneCh <- nil
		}()
		regFunc()
	}()
	// Block until the goroutine is done
	_ = <-doneCh

	if panicMessage == "" {
		t.Errorf("expected a panic, did not get one")
	} else if panicMessage != expectMessage {
		t.Errorf("error mismatch, expect: %q, got: %q", expectMessage, panicMessage)
	}
}

// Test data: methodSet and implementation
type animalMethods struct {
	d dispatcher
}

func (m *animalMethods) Name() string {
	return "animal"
}

func (m *animalMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"cat": {"", ""},
		"dog": {"", ""},
	}
}

type animalParams struct {
	Name string
}

func (m *animalMethods) Cat(ctx context.Context, p *animalParams) (string, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "cat"), p)
	if res, ok := got.(string); ok {
		return res, err
	}
	return "", dispatchReturnError(got, err)
}

func (m *animalMethods) Dog(ctx context.Context, p *animalParams) (string, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "dog"), p)
	if res, ok := got.(string); ok {
		return res, err
	}
	return "", dispatchReturnError(got, err)
}

// Good implementation
type animalImpl struct{}

func (animalImpl) Cat(scp scope, p *animalParams) (string, error) {
	return fmt.Sprintf("%s says meow", p.Name), nil
}

func (animalImpl) Dog(scp scope, p *animalParams) (string, error) {
	return fmt.Sprintf("%s says bark", p.Name), nil
}

// Bad implementation #1 (cat doesn't return enough values)
type badAnimalOneImpl struct{}

func (badAnimalOneImpl) Cat(scp scope, p *animalParams) {
	fmt.Printf("%s says meow", p.Name)
}

func (badAnimalOneImpl) Dog(scp scope, p *animalParams) (string, error) {
	return fmt.Sprintf("%s says bark", p.Name), nil
}

// Bad implementation #2 (dog method name doesn't match)
type badAnimalTwoImpl struct{}

func (badAnimalTwoImpl) Cat(scp scope, p *animalParams) (string, error) {
	return fmt.Sprintf("%s says meow", p.Name), nil
}

func (badAnimalTwoImpl) Doggie(scp scope, p *animalParams) (string, error) {
	return fmt.Sprintf("%s says bark", p.Name), nil
}

// Bad implementation #3 (cat doesn't accept a scope)
type badAnimalThreeImpl struct{}

func (badAnimalThreeImpl) Cat(ctx context.Context, p *animalParams) (string, error) {
	return fmt.Sprintf("%s says meow", p.Name), nil
}

func (badAnimalThreeImpl) Dog(scp scope, p *animalParams) (string, error) {
	return fmt.Sprintf("%s says bark", p.Name), nil
}

// Bad implementation #4 (dog input struct doesn't match)
type badAnimalFourImpl struct{}

func (badAnimalFourImpl) Cat(scp scope, p *animalParams) (string, error) {
	return fmt.Sprintf("%s says meow", p.Name), nil
}

func (badAnimalFourImpl) Dog(scp scope, name string) (string, error) {
	return fmt.Sprintf("%s says bark", name), nil
}

// Bad implementation #5 (dog method is missing)
type badAnimalFiveImpl struct{}

func (badAnimalFiveImpl) Cat(scp scope, p *animalParams) (string, error) {
	return fmt.Sprintf("%s says meow", p.Name), nil
}

// MethodSet with variadic return values
type fruitMethods struct {
	d dispatcher
}

func (m *fruitMethods) Name() string {
	return "fruit"
}

func (m *fruitMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"apple":  {"/apple", "GET"},
		"banana": {"/banana", "GET"},
		"cherry": {"/cherry", "GET"},
		"date":   {"/date", "GET"},
		// entawak cannot be called over RPC
		"entawak": {"", ""},
	}
}

type fruitParams struct {
	Name string
}

func (m *fruitMethods) Apple(ctx context.Context, p *fruitParams) error {
	_, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "apple"), p)
	return err
}

func (m *fruitMethods) Banana(ctx context.Context, p *fruitParams) (string, Cursor, error) {
	got, cur, err := m.d.Dispatch(ctx, dispatchMethodName(m, "banana"), p)
	if res, ok := got.(string); ok {
		return res, cur, err
	}
	return "", nil, dispatchReturnError(got, err)
}

func (m *fruitMethods) Cherry(ctx context.Context, p *fruitParams) error {
	_, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "cherry"), p)
	return err
}

func (m *fruitMethods) Date(ctx context.Context, p *fruitParams) (string, Cursor, error) {
	got, cur, err := m.d.Dispatch(ctx, dispatchMethodName(m, "date"), p)
	if res, ok := got.(string); ok {
		return res, cur, err
	}
	return "", nil, dispatchReturnError(got, err)
}

func (m *fruitMethods) Entawak(ctx context.Context, p *fruitParams) (string, Cursor, error) {
	got, cur, err := m.d.Dispatch(ctx, dispatchMethodName(m, "entawak"), p)
	if res, ok := got.(string); ok {
		return res, cur, err
	}
	return "", nil, dispatchReturnError(got, err)
}

// Implementation for fruit
type fruitImpl struct{}

func (fruitImpl) Apple(scp scope, p *fruitParams) error {
	return fmt.Errorf("no more apples")
}

func (fruitImpl) Banana(scp scope, p *fruitParams) (string, Cursor, error) {
	var cur Cursor
	return "batman", cur, fmt.Errorf("success")
}

func (fruitImpl) Cherry(scp scope, p *fruitParams) error {
	return nil
}

func (fruitImpl) Date(scp scope, p *fruitParams) (string, Cursor, error) {
	var cur Cursor
	return "January 1st", cur, nil
}

func (fruitImpl) Entawak(scp scope, p *fruitParams) (string, Cursor, error) {
	return "mentawa", nil, nil
}
