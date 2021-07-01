package startf

import (
	"strings"
	"testing"
	"github.com/google/go-cmp/cmp"
	"go.starlark.net/syntax"
)

func TestSimpleControlFlow(t *testing.T) {
	controlFlow := makeControlFlowForFunction(t, "testdata/simple.star", "use_branch")

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

func TestMultipleControlFlow(t *testing.T) {
	controlFlow := makeControlFlowForFunction(t, "testdata/simple.star", "branch_multiple")

	actual := controlFlow.stringify()
	expect := `
0: set! a = 1
   set! b = 2
  out: 1
1: if [< a b]
  out: 2,3
2: set! c = [+ b 1]
   set! d = a
   set! e = [+ a b]
  out: 4
3: set! c = [+ a 1]
   print(c)
   set! e = [+ c 2]
  out: 4
4: print([% '%d' e])
  out:
`
	actual = strings.Trim(actual, " \n")
	expect = strings.Trim(expect, " \n")
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestNoElseControlFlow(t *testing.T) {
	controlFlow := makeControlFlowForFunction(t, "testdata/simple.star", "branch_no_else")

	actual := controlFlow.stringify()
	expect := `
0: set! a = 1
   set! b = 2
  out: 1
1: if [< a b]
  out: 2,3
2: set! c = [+ b 1]
   print([% '%d' c])
  out: 3
3: print([% '%d' b])
  out:
`
	actual = strings.Trim(actual, " \n")
	expect = strings.Trim(expect, " \n")
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func makeControlFlowForFunction(t *testing.T, filename, funcname string) *ControlFlow {
	f, err := syntax.Parse(filename, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	funcs, err := collectFunctionDefs(f.Stmts)
	if err != nil {
		t.Fatal(err)
	}
	callGraph := buildCallGraph(funcs)

	fn := callGraph.lookup[funcname]
	body := fn.fn.body

	controlFlow := newControlFlow()
	buildControlFlow(controlFlow, body)
	return controlFlow
}
