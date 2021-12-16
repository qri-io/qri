package staticlark

import (
	"fmt"
)

// analyze the call graph to detect sensitive data being used incorrectly.
// Assume that some functions return sensitive data (such as private datasets)
// while others do dangerous things with data (such as uploading it with http).
// These are declared using the axioms map.
// Traverse the call graph, starting from lowest leaf functions first, and
// for each function, build a control flow graph. Perform dataflow analysis
// on this graph to see which variables are influenced by which values. The
// rule is that sensitive data may not be passed to a dangerous function
func analyzeSensitiveDataflow(graph *callGraph, axioms map[string]*funcNode) ([]Diagnostic, error) {
	dataflowAnalyzer := &dataflowAnalyzer{
		graph:  graph,
		axioms: axioms,
		seen:   make(map[string]struct{}),
	}
	for _, fn := range graph.nodes {
		if err := dataflowAnalyzer.traverseNode(fn); err != nil {
			return nil, err
		}
	}
	return dataflowAnalyzer.diags, nil
}

type dataflowAnalyzer struct {
	graph  *callGraph
	axioms map[string]*funcNode
	seen   map[string]struct{}
	diags  []Diagnostic
	// a stack of sources: values that influence any assignments created
	// within the current control structure
	controlSrcStack [][]string
}

// recursively call this function until all leaf functions are handled
func (da *dataflowAnalyzer) traverseNode(fn *funcNode) error {
	// Only check a given function once
	if _, ok := da.seen[fn.name]; ok {
		return nil
	}
	// Have to check the invoked functions first
	for _, call := range fn.calls {
		if err := da.traverseNode(call); err != nil {
			return err
		}
	}
	// Mark this as being visited
	da.seen[fn.name] = struct{}{}
	// Perhaps it is handled as an axiom
	if satisfiesAxiom(fn, da.axioms) {
		return nil
	}
	return da.analyzeFunction(fn)
}

func satisfiesAxiom(fn *funcNode, axioms map[string]*funcNode) bool {
	fname := fn.name
	if axioms == nil {
		return false
	}
	if lookup, ok := axioms[fname]; ok {
		fn.dangerousParams = lookup.dangerousParams
		fn.sensitiveReturn = lookup.sensitiveReturn
		fn.reasonParams = lookup.reasonParams
		return true
	}
	return false
}

func (da *dataflowAnalyzer) analyzeFunction(fn *funcNode) error {
	fname := fn.name
	f := da.graph.lookup[fname]
	if f == nil {
		return fmt.Errorf("showing control flow, function %q not found", fname)
	}
	params := f.params

	controlFlow, err := newControlFlowFromFunc(f)
	if err != nil {
		return err
	}

	env := newEnvironment()
	env.markParams(params)

	if err := da.analyzeSequence(0, len(controlFlow.blocks), controlFlow, env, fn); err != nil {
		return err
	}

	dangerousParams, reasonParams := env.getHighSensitive(params)
	fn.dangerousParams = dangerousParams
	fn.reasonParams = reasonParams
	return nil
}

// analyze the sequence of blocks beginning at start, until finish
func (da *dataflowAnalyzer) analyzeSequence(start, finish int, cf *controlFlow, env *environment, fn *funcNode) error {

	index := start
	for index >= 0 && index < finish {
		block := cf.blocks[index]

		if block.isLinear() {
			// linear flow simply analyzes the block, then follows the edge
			if err := da.analyzeBlock(block, env, fn); err != nil {
				return err
			}

			if len(block.edges) == 1 {
				index = block.edges[0]
			} else {
				break
			}

		} else if block.isIfCondition() {
			// if statements will analyze both true and false branches, then union
			// the environments from each
			trueIdx := block.edges[0]
			falseIdx := block.edges[1]
			joinIdx := block.join

			trueEnv := env.clone()
			falseEnv := env.clone()

			// the data sources for the if condition will influence any assignments
			// that happen in either branch
			da.pushControlBindings(block.units[0].DataSources())

			if err := da.analyzeSequence(trueIdx, joinIdx, cf, trueEnv, fn); err != nil {
				return err
			}
			if err := da.analyzeSequence(falseIdx, joinIdx, cf, falseEnv, fn); err != nil {
				return err
			}

			da.popControlBindings()

			env.copyFrom(trueEnv.union(falseEnv))
			index = joinIdx

		} else {
			// TODO(dustmop): Handle loops also - run the loop repeatedly
			// until the environment reaches a fixed-point
			return fmt.Errorf("TODO: block type %v not implemented", block)
		}
	}

	return nil
}

func (da *dataflowAnalyzer) analyzeBlock(block *codeBlock, env *environment, fn *funcNode) error {
	fname := fn.name
	// iterate each unit. Could be an assignment, or function call, etc
	for _, unit := range block.units {

		// get data sources that are not function calls
		sources := []string{}
		for _, src := range unit.DataSources() {
			if _, ok := da.graph.lookup[src]; !ok {
				sources = append(sources, src)
			}
		}

		invokes := unit.Invocations()
		if dest := unit.AssignsTo(); dest != "" {
			// if this variable is being assigned the output of a function
			// that returns secret data, mark it as secret itself
			for _, inv := range invokes {
				if fn, ok := da.graph.lookup[inv.Name]; ok && fn.sensitiveReturn {
					sources = append(sources, sensitiveVarName)
				}
			}
			// assign the data sources to this variable
			sources = append(sources, da.controlSources()...)
			env.assign(dest, sources)
		}

		if unit.IsReturn() {
			// if the unit is a return statement, which is returning secret
			// data, mark the return of this function as being sensitive
			for _, src := range unit.DataSources() {
				_, ok := da.graph.lookup[src]
				if !ok {
					if env.isSecret(src) {
						fn.sensitiveReturn = true
					}
				}
			}
		}

		// check if any secret data is being passed to sensitive function
		// arguments
		for _, inv := range invokes {
			// skip built-in functions and control structures
			if da.builtinOrControlStructure(inv.Name) {
				continue
			}

			fn, ok := da.graph.lookup[inv.Name]
			if !ok {
				return fmt.Errorf("invoked function %s not found", inv.Name)
			}

			for i, arg := range inv.Args {
				danger := fn.dangerousParams
				if danger != nil && i < len(danger) && danger[i] {
					// receiving param can potentially be dangerous
					if env.isSecret(arg) {
						// secret being sent to dangerous param
						prev := fn.reasonParams[i]
						msg := fmt.Sprintf("secrets may leak, variable %s is secret\n%s", arg, prev.String())
						d := Diagnostic{
							Pos:      unit.where,
							Category: "leak",
							Message:  msg,
						}
						da.diags = append(da.diags, d)
					}
					// taint vars so that the sources become dangerous
					prev := fn.reasonParams[i]
					reason := makeReason(unit.where, fname, arg, inv.Name, fn.params[i], prev)
					env.taint(arg, reason)
				}
			}
		}
	}

	return nil
}

func (da *dataflowAnalyzer) builtinOrControlStructure(name string) bool {
	return name == "if"
}

func (da *dataflowAnalyzer) pushControlBindings(sources []string) {
	da.controlSrcStack = append(da.controlSrcStack, sources)
}

func (da *dataflowAnalyzer) popControlBindings() {
	lastIndex := len(da.controlSrcStack) - 1
	da.controlSrcStack = da.controlSrcStack[:lastIndex]
}

func (da *dataflowAnalyzer) controlSources() []string {
	result := make([]string, 0)
	for _, row := range da.controlSrcStack {
		result = append(result, row...)
	}
	return result
}
