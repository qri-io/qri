package startf

import (
	"strings"
	"testing"
	"github.com/google/go-cmp/cmp"
	"go.starlark.net/syntax"
)

func TestSimpleControlFlow(t *testing.T) {
	filename := "testdata/simple.star"
	f, err := syntax.Parse(filename, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	funcs, err := collectFunctionDefs(f.Stmts)
	if err != nil {
		t.Fatal(err)
	}
	callGraph := buildCallGraph(funcs)

	fn := callGraph.lookup["use_branch"]
	body := fn.fn.body

	controlFlow := newControlFlow()
	buildControlFlow(controlFlow, body)

	actual := controlFlow.stringify()
	expect := `
0: set! a = 1
   set! b = 2
  out: 1
1: if [< a b]
  out: 2,3
2: set! c = [+ b 1]
  out: 4
3: set! c = [+ a 1]
  out: 4
4: print([% '%d' c])
  out:
`
	actual = strings.Trim(actual, " \n")
	expect = strings.Trim(expect, " \n")
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
