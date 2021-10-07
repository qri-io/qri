package staticlark

import (
	"fmt"
	"strings"

	"go.starlark.net/syntax"
)

// ShowAnalysis performs static analysis and prints the results to stdout
func ShowAnalysis(filename string) {
	err := doAnalyze(filename)
	if err != nil {
		fmt.Printf("analyzer error: %s\n", err)
	}
}

// parse script, collect function definitions, build a call graph, then display it
func doAnalyze(filename string) error {
	f, err := syntax.Parse(filename, nil, 0)
	if err != nil {
		return err
	}
	funcs, topLevel, err := collectFuncDefsTopLevelCalls(f.Stmts)
	if err != nil {
		return err
	}

	// Build a graph of all calls, using top level calls
	callGraph := buildCallGraph(funcs, topLevel)

	unused := callGraph.findUnusedFuncs()
	if len(unused) > 0 {
		fmt.Printf("Functions not called: %v\n", strings.Join(unused, " "))
	}
	return nil
}
