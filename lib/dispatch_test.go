package lib

import (
	"context"
	"fmt"
	"testing"
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

// Implementation for fruit

type fruitImpl struct{}

func (fruitImpl) Apple(scp scope, p *fruitParams) error {
	return fmt.Errorf("no more apples")
}

func (fruitImpl) Banana(scp scope, p *fruitParams) (string, Cursor, error) {
	var cur Cursor
	return "batman", cur, fmt.Errorf("success")
}
