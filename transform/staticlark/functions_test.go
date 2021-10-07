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
		text = fmt.Sprintf("%s%d: %s %v\n", text, i, f.name, f.calls)
	}

	expect := `0: use_branch [print]
1: branch_multiple [print print]
2: branch_no_else [print print]
3: branch_nested [print print]
4: top_level_func [len branch_multiple branch_no_else another_function]
5: another_function [branch_nested branch_no_else]
`
	if diff := cmp.Diff(expect, text); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
