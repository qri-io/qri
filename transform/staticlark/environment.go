package staticlark

import (
	"fmt"
	"path"
	"sort"
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

func (e *environment) String() string {
	result := "Env:\n"
	keys := getKeys(e.vars)
	for _, name := range keys {
		der := e.vars[name]
		status := ""
		if _, ok := e.taints[name]; ok {
			status = " tainted"
		}
		result += fmt.Sprintf("  %s -> %s%s\n", name, der, status)
	}
	return result
}

func (e *environment) clone() *environment {
	vars := make(map[string]*deriv, len(e.vars))
	taints := make(map[string]reason, len(e.taints))
	for name := range e.vars {
		vars[name] = e.vars[name].clone()
	}
	for name := range e.taints {
		taints[name] = e.taints[name].clone()
	}
	return &environment{vars: vars, taints: taints}
}

func (e *environment) union(other *environment) *environment {
	result := e.clone()
	for name := range other.vars {
		otherDerivs := other.vars[name]
		result.vars[name].merge(otherDerivs)
		// TODO(dustmop): union taints also
	}
	return result
}

func (e *environment) copyFrom(other *environment) {
	for name := range other.vars {
		e.vars[name] = other.vars[name]
	}
}

func getKeys(m map[string]*deriv) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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

func (d *deriv) String() string {
	return strings.Join(d.inputs, ",")
}

func (d *deriv) clone() *deriv {
	inputs := make([]string, len(d.inputs))
	for i, inp := range d.inputs {
		inputs[i] = inp
	}
	return &deriv{inputs: inputs, param: d.param}
}

func (d *deriv) merge(other *deriv) {
	for _, val := range other.inputs {
		if !arrayContains(d.inputs, val) {
			d.inputs = append(d.inputs, val)
		}
	}
	if other.param {
		d.param = true
	}
}

func arrayContains(arr []string, val string) bool {
	for _, elem := range arr {
		if elem == val {
			return true
		}
	}
	return false
}

type reason struct {
	lines []string
}

func (r reason) clone() reason {
	lines := make([]string, len(r.lines))
	for i, ln := range r.lines {
		lines[i] = ln
	}
	return reason{lines: lines}
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
