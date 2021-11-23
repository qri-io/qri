package staticlark

import (
	"fmt"
	"path"
	"strings"

	"go.starlark.net/syntax"
)

const sensitiveVarName = "$"

// environment is a collection of variable bindings, and the
// list of other bindings that it derives value from
// TODO(dustmop): Handle lexical scoping
type environment struct {
	vars   map[string]*deriv
	taints map[string]reason
}

func newEnvironment() *environment {
	return &environment{
		vars:   make(map[string]*deriv),
		taints: make(map[string]reason),
	}
}

func (e *environment) markParams(params []string) {
	for _, p := range params {
		e.vars[p] = &deriv{inputs: []string{p}, param: true}
	}
}

func (e *environment) assign(dest string, sources []string) {
	result := []string{}
	for _, src := range sources {
		if src == sensitiveVarName {
			// meta variable, don't try to resolve
			result = append(result, src)
			continue
		}
		if lookup, ok := e.vars[src]; ok {
			result = append(result, lookup.inputs...)
		} else {
			result = append(result, src)
		}
	}
	if e.vars[dest] == nil {
		e.vars[dest] = &deriv{}
	}
	e.vars[dest].inputs = result
}

func (e *environment) isSecret(name string) bool {
	if e.vars[name] == nil {
		return false
	}
	for _, inp := range e.vars[name].inputs {
		if inp == sensitiveVarName {
			return true
		}
	}
	return false
}

func (e *environment) taint(name string, reason reason) {
	if lookup, ok := e.vars[name]; ok {
		// variable used
		for _, inp := range lookup.inputs {
			e.taints[inp] = reason
		}
	} else {
		// parameter used direcly
		e.taints[name] = reason
	}
}

func (e *environment) getHighSensitive(params []string) ([]bool, []reason) {
	dangerousParams := make([]bool, len(params))
	reasonParams := make([]reason, len(params))
	for i, p := range params {
		if reason, ok := e.taints[p]; ok {
			dangerousParams[i] = true
			reasonParams[i] = reason
		}
	}
	return dangerousParams, reasonParams
}

type deriv struct {
	inputs []string
	param  bool
}

type reason struct {
	lines []string
}

func makeReason(pos syntax.Position, currFunc, varName, invokeName, paramName string, prev reason) reason {
	baseName := path.Base(pos.Filename())
	text := fmt.Sprintf("%s:%d: %s passes %s to %s argument %s", baseName, pos.Line, currFunc, varName, invokeName, paramName)
	return reason{
		lines: append([]string{text}, prev.lines...),
	}
}

func (r reason) String() string {
	return strings.Join(r.lines, "\n")
}
