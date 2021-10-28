package staticlark

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.starlark.net/syntax"
)

func TestCallGraph(t *testing.T) {
	filename := "testdata/some_funcs.star"

	f, err := syntax.Parse(filename, nil, 0)
	if err != nil {
		t.Error(err)
	}
	funcs, topLevel, err := collectFuncDefsTopLevelCalls(f.Stmts)
	if err != nil {
		t.Error(err)
	}

	// Build a graph of all calls, Detect unused functions
	callGraph := buildCallGraph(funcs, topLevel)

	actual := callGraph.String()
	expect := `print
use_branch
 print
branch_multiple
 print
branch_no_else
 print
branch_nested
 print
another_function
 branch_nested
  print
 branch_no_else
  print
top_level_func
 use_branch
  print
 branch_multiple
  print
 branch_no_else
  print
 another_function
  branch_nested
   print
  branch_no_else
   print
branch_elses
 print
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUnusedFunctions(t *testing.T) {
	filename := "testdata/more_funcs.star"

	f, err := syntax.Parse(filename, nil, 0)
	if err != nil {
		t.Error(err)
	}
	funcs, topLevel, err := collectFuncDefsTopLevelCalls(f.Stmts)
	if err != nil {
		t.Error(err)
	}

	// Build a graph of all calls, Detect unused functions
	callGraph := buildCallGraph(funcs, topLevel)

	unused := callGraph.findUnusedFuncs()
	expectUnused := []Diagnostic{
		{Category: "unused", Message: "func_c"},
		{Category: "unused", Message: "func_e"},
		{Category: "unused", Message: "func_f"},
	}
	if diff := cmp.Diff(expectUnused, unused, cmpopts.IgnoreFields(Diagnostic{}, "Pos")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
