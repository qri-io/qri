package staticlark

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.starlark.net/syntax"
)

func TestCollectFunctions(t *testing.T) {
	filename := "testdata/some_funcs.star"

	f, err := syntax.Parse(filename, nil, 0)
	if err != nil {
		t.Error(err)
	}
	funcs, _, err := collectFuncDefsTopLevelCalls(f.Stmts)
	if err != nil {
		t.Error(err)
	}

	text := ""
	for i, f := range funcs {
		text = fmt.Sprintf("%s%d: %s %v\n", text, i, f.name, f.callNames)
	}

	expect := `0: use_branch [print]
1: branch_multiple [print print]
2: branch_no_else [print print]
3: branch_nested [print print]
4: top_level_func [use_branch len branch_multiple branch_no_else another_function]
5: another_function [branch_nested branch_no_else]
6: branch_elses [print print print print print]
`
	if diff := cmp.Diff(expect, text); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestCollectFunctionsAllSyntax(t *testing.T) {
	filename := "testdata/all_syntax.star"

	f, err := syntax.Parse(filename, nil, 0)
	if err != nil {
		t.Error(err)
	}
	funcs, _, err := collectFuncDefsTopLevelCalls(f.Stmts)
	if err != nil {
		t.Error(err)
	}

	text := ""
	for i, f := range funcs {
		text = fmt.Sprintf("%s%d: %s %v\n", text, i, f.name, f.callNames)
	}

	// TODO(dustmop): fact_iter is the name of 2 different inner functions
	// Should also appear in this list
	expect := `0: some_vals [add_one]
1: add_one []
2: mult []
3: half []
4: calc [fact_iter]
5: yet_more [math.floor fact_iter]
6: do_math [some_vals calc yet_more print]
`
	if diff := cmp.Diff(expect, text); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
