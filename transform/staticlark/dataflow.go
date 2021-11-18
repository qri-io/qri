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
func analyzeSensitiveDataflow(graph *callGraph, axioms map[string]*funcNode) error {
	seen := map[string]struct{}{}
	for _, fn := range graph.nodes {
		if err := dataflowTraverseNode(fn, graph, axioms, seen); err != nil {
			return err
		}
	}
	return nil
}

// recursively call this function until all leaf functions are handled
func dataflowTraverseNode(fn *funcNode, graph *callGraph, axioms map[string]*funcNode, seen map[string]struct{}) error {
	// Only check a given function once
	if _, ok := seen[fn.name]; ok {
		return nil
	}
	// Have to check the invoked functions first
	for _, call := range fn.calls {
		if err := dataflowTraverseNode(call, graph, axioms, seen); err != nil {
			return err
		}
	}
	// Mark this as being visited
	seen[fn.name] = struct{}{}
	// Perhaps it is handled as an axiom
	if satisfiesAxiom(fn, axioms) {
		return nil
	}
	return dataflowAnalyzeFunction(fn, graph)
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

func dataflowAnalyzeFunction(fn *funcNode, graph *callGraph) error {
	fname := fn.name
	f := graph.lookup[fname]
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
				_, ok := graph.lookup[src]
				if !ok {
					sources = append(sources, src)
				}
			}

			invokes := unit.Invocations()
			if dest := unit.AssignsTo(); dest != "" {
				// If this variable is being assigned the output of a function
				// that returns secret data, mark it as secret itself
				for _, inv := range invokes {
					fn, ok := graph.lookup[inv.Name]
					if ok && fn.sensitiveReturn {
						sources = append(sources, sensitiveVarName)
					}
				}
				// Assign the data sources to this variable
				env.assign(dest, sources)
			}

			// Check if any secret data is being passed to sensitive function
			// arguments
			for _, inv := range invokes {
				fn, ok := graph.lookup[inv.Name]
				if !ok {
					return fmt.Errorf("invoked function %s not found", inv.Name)
				}

				for i, arg := range inv.Args {
					if fn.dangerousParams != nil && i < len(fn.dangerousParams) && fn.dangerousParams[i] {
						if env.isSecret(arg) {
							// TODO(dustmop): Instead, accumulate diagnostic messages, using
							// the Diagnostic type in analyze.go, intsead of returning an
							// error
							return fmt.Errorf("secrets may leak, variable %s is secret", arg)
						}
						// Taint vars so that the sources become dangerous
						prev := fn.reasonParams[i]
						reason := makeReason(unit.where, fname, arg, inv.Name, params[i], prev)
						env.taint(arg, reason)
					}
				}
			}
		}
	}

	dangerousParams, reasonParams := env.getHighSensitive(params)
	fn.dangerousParams = dangerousParams
	fn.reasonParams = reasonParams
	// TODO(dustmop): assign sensitiveReturn
	return nil
}
