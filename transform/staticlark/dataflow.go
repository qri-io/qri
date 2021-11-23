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

func (da *dataflowAnalyzer) analyzeFunction(analyzing *funcNode) error {
	fname := analyzing.name
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

	// TODO(dustmop): This only handles linear control flow. Need to handle
	// branch and loops also.
	// * Branch: run this over each branch, union the results
	// * Loop: run the loop repeatedly until a fixed-point is reached
	for _, block := range controlFlow.blocks {

		// Iterate each unit, an assignment, or function call, etc
		for _, unit := range block.units {

			// Get data sources that are not function calls
			sources := []string{}
			for _, src := range unit.DataSources() {
				_, ok := da.graph.lookup[src]
				if !ok {
					sources = append(sources, src)
				}
			}

			invokes := unit.Invocations()
			if dest := unit.AssignsTo(); dest != "" {
				// If this variable is being assigned the output of a function
				// that returns secret data, mark it as secret itself
				for _, inv := range invokes {
					fn, ok := da.graph.lookup[inv.Name]
					if ok && fn.sensitiveReturn {
						sources = append(sources, sensitiveVarName)
					}
				}
				// Assign the data sources to this variable
				env.assign(dest, sources)
			}

			if unit.IsReturn() {
				for _, src := range unit.DataSources() {
					_, ok := da.graph.lookup[src]
					if !ok {
						if env.isSecret(src) {
							analyzing.sensitiveReturn = true
						}
					}
				}
			}

			// Check if any secret data is being passed to sensitive function
			// arguments
			for _, inv := range invokes {
				fn, ok := da.graph.lookup[inv.Name]
				if !ok {
					return fmt.Errorf("invoked function %s not found", inv.Name)
				}

				for i, arg := range inv.Args {
					danger := fn.dangerousParams
					if danger != nil && i < len(danger) && danger[i] {
						if env.isSecret(arg) {
							prev := fn.reasonParams[i]
							d := Diagnostic{
								Pos:      unit.where,
								Category: "leak",
								Message:  fmt.Sprintf("secrets may leak, variable %s is secret\n%s", arg, prev.String()),
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
	}

	dangerousParams, reasonParams := env.getHighSensitive(params)
	analyzing.dangerousParams = dangerousParams
	analyzing.reasonParams = reasonParams
	return nil
}
