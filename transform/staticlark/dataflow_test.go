package staticlark

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func TestSingleFunctionDataflow(t *testing.T) {
	filename := "testdata/safety_funcs.star"
	callGraph := mustBuildCallGraphFromFile(t, filename)

	// axioms: get_secret returns a sensitive value, and dangerous does
	// something unsafe with its first parameter
	axioms := map[string]*funcNode{
		"get_secret": &funcNode{
			name: "get_secret",
			sensitiveReturn: true,
		},
		"dangerous": &funcNode{
			dangerousParams: []bool{true, false},
			reasonParams: []reason{
				reason{lines: []string{"assume it is dangerous"}},
				reason{},
			},
		},
	}

	diags, err := analyzeSensitiveDataflow(callGraph, axioms)
	if err != nil {
		t.Error(err)
	}

	expectDiags := []Diagnostic{
		Diagnostic{
			Pos:      syntax.MakePosition(&filename, 22, 3),
			Category: "leak",
			Message:  "secrets may leak, variable c is secret\nassume it is dangerous",
		},
	}

	ignoreCmp := cmpopts.IgnoreUnexported(syntax.Position{})
	if diff := cmp.Diff(expectDiags, diags, ignoreCmp); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	fn := callGraph.lookup["processor"]
	if !fn.sensitiveReturn {
		t.Errorf("expected: function `processor` returns a sensitive value")
	}
}

func TestCallTraceDataflow(t *testing.T) {
	filename := "testdata/call_trace.star"
	callGraph := mustBuildCallGraphFromFile(t, filename)

	// axioms: get_secret returns a sensitive value, and dangerous does
	// something unsafe with its first parameter
	axioms := map[string]*funcNode{
		"get_secret": &funcNode{
			name: "get_secret",
			sensitiveReturn: true,
		},
		"dangerous": &funcNode{
			dangerousParams: []bool{true, false},
			reasonParams: []reason{
				reason{lines: []string{"assume it is dangerous"}},
				reason{},
			},
		},
	}

	diags, err := analyzeSensitiveDataflow(callGraph, axioms)
	if err != nil {
		t.Error(err)
	}

	expectDiags := []Diagnostic{
		Diagnostic{
			Pos:      syntax.MakePosition(&filename, 33, 3),
			Category: "leak",
		Message: `secrets may leak, variable i is secret
call_trace.star:26: middle passes f to bottom argument m
call_trace.star:17: bottom passes b to dangerous argument s
assume it is dangerous`,
		},
		Diagnostic{
			Pos:      syntax.MakePosition(&filename, 33, 3),
			Category: "leak",
		Message: `secrets may leak, variable k is secret
call_trace.star:26: middle passes g to bottom argument n
call_trace.star:18: bottom passes n to dangerous argument s
assume it is dangerous`,
		},
	}

	ignoreCmp := cmpopts.IgnoreUnexported(syntax.Position{})
	if diff := cmp.Diff(expectDiags, diags, ignoreCmp); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func mustBuildCallGraphFromFile(t *testing.T, filename string) *callGraph {
	f, err := syntax.Parse(filename, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	funcs, topLevel, err := collectFuncDefsTopLevelCalls(f.Stmts)
	if err != nil {
		t.Fatal(err)
	}
	globals := newSymtable(starlark.Universe)
	return buildCallGraph(funcs, topLevel, globals)
}
