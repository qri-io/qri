package staticlark

import (
	golog "github.com/ipfs/go-log"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

var log = golog.Logger("staticlark")

// AnalyzeFile performs static analysis and returns diagnostic results
func AnalyzeFile(filename string) ([]Diagnostic, error) {
	// Parse the script to abstract syntax
	f, err := syntax.Parse(filename, nil, 0)
	if err != nil {
		return nil, err
	}
	// Collect function definitions and top level function calls
	funcs, topLevel, err := collectFuncDefsTopLevelCalls(f.Stmts)
	if err != nil {
		return nil, err
	}
	// Constuct pre-defined global symbols
	globals := newSymtable(starlark.Universe)
	// Build a graph of all calls, using top level calls and pre-defined globals
	callGraph := buildCallGraph(funcs, topLevel, globals)

	// Trace sensitive data using dataflow analysis
	dataflowDiags, err := analyzeSensitiveDataflow(callGraph, nil)
	if err != nil {
		return nil, err
	}

	// Return any unused functions
	// TODO(dustmop): As more analysis steps are introduced, refactor this
	// into a generic interface that creates Diagnostics
	unusedDiags := callGraph.findUnusedFuncs()
	return append(dataflowDiags, unusedDiags...), nil
}

// Diagnostic represents a diagnostic message describing an issue with the code
type Diagnostic struct {
	Pos      syntax.Position
	Category string
	Message  string
}

func newSymtable(symbols starlark.StringDict) map[string]*funcNode {
	table := make(map[string]*funcNode)
	for name := range symbols {
		table[name] = &funcNode{name: name}
	}
	return table
}
