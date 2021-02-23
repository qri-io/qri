package lib

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// Cursor provieds pagination details for requests
// TODO(b5): finish this interface
type Cursor interface {
	Params() interface{}
}

type cursor struct {
	params interface{}
}

func (c cursor) Params() interface{} { return c.params }

func newCursor(params interface{}) Cursor {
	return cursor{params: params}
}

var DispatchError = fmt.Errorf("dispatch:")

type Dispatch struct {
	inst *Instance
	tree dispatchContainerNode
	// methods map[APIEndpoint]reflect.Value
}

func NewDispatch(inst *Instance) *Dispatch {
	methodSets := Receivers(inst)
	tree := newContainerNode("")
	for _, methodSet := range methodSets {
		tree.AddChild(methodSet.dispatchTree())
	}

	fmt.Printf("dispatch method tree:\n%s\n", dispatchTreeString(tree))

	return &Dispatch{
		inst: inst,
		// methods: methods,
		tree: tree,
	}
}

func (d *Dispatch) NewMethodInputParams(methodPath ...string) (res interface{}, err error) {
	methodNode, err := selectMethodNode(d.tree, methodPath)
	if err != nil {
		return nil, err
	}
	return methodNode.NewInputParam(), nil
}

func (d *Dispatch) Call(ctx context.Context, params interface{}, methodPath ...string) (res interface{}, c Cursor, err error) {
	log.Debugw("calling dispatch method", "path", methodPath, "params", params)
	methodNode, err := selectMethodNode(d.tree, methodPath)
	if err != nil {
		return nil, nil, err
	}

	response := methodNode.Method().Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(params),
	})
	switch len(response) {
	case 1:
		err, ok := response[0].Interface().(error)
		if !ok && !response[0].IsNil() {
			return nil, nil, fmt.Errorf("%w return value must be an error type, got: %s", DispatchError, response[0].Kind())
		}
		return nil, nil, err
	case 2:
		res = response[0].Interface()
		err, ok := response[1].Interface().(error)
		if !ok && !response[1].IsNil() {
			return nil, nil, fmt.Errorf("%w second return value must be an error type, got: %s", DispatchError, response[1].Kind())
		}
		return res, nil, err
	case 3:
		res = response[0].Interface()
		c, ok := response[1].Interface().(Cursor)
		if !ok {
			return nil, nil, fmt.Errorf("%w second return value in three argument return must be a Cursor type, got: %s %#v", DispatchError, response[1].Kind(), c)
		}
		err, ok := response[2].Interface().(error)
		if !ok && !response[2].IsNil() {
			return nil, nil, fmt.Errorf("%w third return value must be an error type, got: %s", DispatchError, response[2].Kind())
		}
		return res, c, err

	default:
		return nil, nil, fmt.Errorf("%w methods must return either 1, 2 or 3 values. method %q returned %d", DispatchError, strings.Join(methodPath, ", "), len(response))
	}
}

type dispatchNode interface {
	Name() string
	Children() []dispatchNode
	Child(name string) (dispatchNode, bool)
}

type dispatchContainerNode interface {
	dispatchNode
	AddChild(dispatchNode) error
}

type dispatchMethodNode interface {
	dispatchNode
	NewInputParam() interface{}
	Method() reflect.Value
}

func selectMethodNode(tree dispatchNode, methodPath []string) (dispatchMethodNode, error) {
	var (
		node = tree
		ok   bool
	)
	for _, name := range methodPath {
		ch, ok := node.Child(name)
		if !ok {
			return nil, fmt.Errorf("%s tree node %q has no child method named %q", DispatchError, node.Name(), name)
		}
		node = ch
	}

	method, ok := node.(dispatchMethodNode)
	if !ok {
		return nil, fmt.Errorf("%w call path %q does not terminate with a method", DispatchError, strings.Join(methodPath, ", "))
	}

	return method, nil
}

type containerNode struct {
	name    string
	chNames []string
	ch      map[string]dispatchNode
}

func newContainerNode(name string, children ...dispatchNode) dispatchContainerNode {
	names := make([]string, 0, len(children))
	ch := make(map[string]dispatchNode)
	for _, n := range children {
		names = append(names, n.Name())
		ch[n.Name()] = n
	}

	return &containerNode{
		name:    name,
		chNames: names,
		ch:      ch,
	}
}

func (n containerNode) Name() string { return n.name }
func (n containerNode) Children() []dispatchNode {
	children := make([]dispatchNode, len(n.chNames))
	for i, name := range n.chNames {
		children[i] = n.ch[name]
	}
	return children
}
func (n *containerNode) Child(name string) (dispatchNode, bool) {
	ch, ok := n.ch[name]
	return ch, ok
}

func (n *containerNode) AddChild(ch dispatchNode) error {
	n.chNames = append(n.chNames, ch.Name())
	// if a child with this name already exists, merge trees
	if existingChild, exists := n.ch[ch.Name()]; exists {
		cont, ok := existingChild.(dispatchContainerNode)
		if !ok {
			return fmt.Errorf("%w child named %q already exists & cannot add children", DispatchError, existingChild.Name())
		}
		for _, desc := range ch.Children() {
			if err := cont.AddChild(desc); err != nil {
				return err
			}
		}
		return nil
	}

	n.ch[ch.Name()] = ch
	return nil
}

type methodNode struct {
	name     string
	newParam func() interface{}
	method   reflect.Value
}

var _ dispatchMethodNode = (*methodNode)(nil)

func newMethodNode(name string, method interface{}) dispatchNode {
	methodV := reflect.ValueOf(method)
	if methodV.Kind() != reflect.Func {
		panic("newMethodNode method argument must be a function")
	}
	argElem := methodV.Type().In(1).Elem()

	return methodNode{
		name:   name,
		method: methodV,
		newParam: func() interface{} {
			return reflect.New(argElem).Interface()
		},
	}
}

func (n methodNode) Name() string                           { return n.name }
func (n methodNode) Children() []dispatchNode               { return nil }
func (n methodNode) Child(name string) (dispatchNode, bool) { return nil, false }
func (n methodNode) NewInputParam() interface{}             { return n.newParam() }
func (n methodNode) Method() reflect.Value                  { return n.method }

func walk(node dispatchNode, depth int, visit func(n dispatchNode, depth int)) {
	visit(node, depth)
	for _, ch := range node.Children() {
		walk(ch, depth+1, visit)
	}
}

func dispatchTreeString(tree dispatchNode) string {
	str := ""
	visit := func(n dispatchNode, depth int) {
		str += fmt.Sprintf("%s%s\n", strings.Repeat("  ", depth), n.Name())
	}
	walk(tree, 0, visit)
	return str
}
