package staticlark

import (
	"go.starlark.net/syntax"
)

// AnalyzeFile performs static analysis and results diagnostic results
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
	// Build a graph of all calls, using top level calls
	callGraph := buildCallGraph(funcs, topLevel)
	// Return any unused functions
	// TODO(dustmop): As more analysis steps are introduced, refactor this
	// into a generic interface that creates Diagnostics
	return callGraph.findUnusedFuncs(), nil
}

// Diagnostic represents a diagnostic message describing an issue with the code
type Diagnostic struct {
	Pos      syntax.Position
	Category string
	Message  string
}
