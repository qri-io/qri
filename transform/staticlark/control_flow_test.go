package staticlark

import (
	"github.com/google/go-cmp/cmp"
	"go.starlark.net/syntax"
	"testing"
)

func TestControlFlowIf(t *testing.T) {
	// a simple function with an if/else
	funcmap := mustReadScriptFunctionMap(t, "testdata/some_funcs.star")
	cf, err := newControlFlowFromFunc(funcmap["use_branch"])
	if err != nil {
		t.Fatal(err)
	}

	expect := `0: [set! a 1]
   [set! b 2]
  out: 1
1: [if [< a b]]
  out: 2,3, join: 4
2: [set! c [+ b 1]]
  out: 4
3: [set! c [+ a 1]]
  out: 4
4: [print [% '%d' c]]
  out: -
`
	actual := cf.stringify()
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// a function with if, but no else
	cf, err = newControlFlowFromFunc(funcmap["branch_no_else"])
	if err != nil {
		t.Fatal(err)
	}

	expect = `0: [set! a 1]
   [set! b 2]
  out: 1
1: [if [< a b]]
  out: 2,3, join: 3
2: [set! c [+ b 1]]
   [print [% '%d' c]]
  out: 3
3: [print [% '%d' b]]
  out: -
`

	actual = cf.stringify()
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// a function with if nested within an if
	cf, err = newControlFlowFromFunc(funcmap["branch_nested"])
	if err != nil {
		t.Fatal(err)
	}

	expect = `0: [set! a 1]
   [set! b 2]
  out: 1
1: [if [< a b]]
  out: 2,5, join: 6
2: [set! c [+ b 1]]
   [set! d a]
  out: 3
3: [if [> d c]]
  out: 4,6, join: 6
4: [set! c [+ d 2]]
  out: 6
5: [set! c [+ a 1]]
   [print c]
   [set! e [+ c 2]]
  out: 6
6: [print [% '%d' e]]
  out: -
`
	actual = cf.stringify()
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// a function with if and elif and else
	funcmap = mustReadScriptFunctionMap(t, "testdata/some_funcs.star")
	cf, err = newControlFlowFromFunc(funcmap["branch_elses"])
	if err != nil {
		t.Fatal(err)
	}

	expect = `0: [set! a 1]
   [set! b 2]
  out: 1
1: [if [< a b]]
  out: 2,8, join: 9
2: [set! c [+ b 1]]
  out: 3
3: [if [< c 1]]
  out: 4,5, join: 9
4: [print 'small']
  out: 9
5: [if [< c 5]]
  out: 6,7, join: 9
6: [print 'medium']
  out: 9
7: [print 'large']
  out: 9
8: [print 'ok']
  out: 9
9: [print 'done']
  out: -
`
	actual = cf.stringify()
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestControlFlowSimpleLoop(t *testing.T) {
	funcmap := mustReadScriptFunctionMap(t, "testdata/loop_funcs.star")

	cf, err := newControlFlowFromFunc(funcmap["stddev"])
	if err != nil {
		t.Fatal(err)
	}

	expect := `0: [set! total 0]
  out: 1
1: [for x ls]
  out: 2,3
2: [set! total [+= total x]]
  out: 1
3: [set! n [len ls]]
   [set! mean [/ total n]]
   [set! result 0]
  out: 4
4: [for x ls]
  out: 5,6
5: [set! diff [- x mean]]
   [set! result [+= result [* diff diff]]]
  out: 4
6: [set! variance [/ result n]]
   [return [math.sqrt variance]]
  out: return
`
	actual := cf.stringify()
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestControlFlowLoopWithBreak(t *testing.T) {
	funcmap := mustReadScriptFunctionMap(t, "testdata/loop_funcs.star")

	cf, err := newControlFlowFromFunc(funcmap["gcd_debug"])
	if err != nil {
		t.Fatal(err)
	}

	// TODO(dustmop): `break` should instead be 9
	expect := `0: [print "gcd starting"]
  out: 1
1: [for n [range 20]]
  out: 2,9
2: [print "gcd a = %d, b = %d" a b]
  out: 3
3: [if [== a b]]
  out: 4,5
4: [print "gcd break at step %d" n]
   [break]
  out: break
5: [print "still going"]
  out: 6
6: [if [> a b]]
  out: 7,8
7: [set! a [- a b]]
  out: 1
8: [set! b [- b a]]
  out: 1
9: [print "gcd returns %d" a]
   [return a]
  out: return
`
	actual := cf.stringify()
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func mustReadScriptFunctionMap(t *testing.T, filename string) map[string]*funcNode {
	f, err := syntax.Parse(filename, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Collect function definitions and top level function calls
	funcs, _, err := collectFuncDefsTopLevelCalls(f.Stmts)
	if err != nil {
		t.Fatal(err)
	}
	fmap := make(map[string]*funcNode)
	for _, f := range funcs {
		fmap[f.name] = f
	}
	return fmap
}

func TestUnitBasic(t *testing.T) {
	root := unit{
		atom: "set!",
		tail: []*unit{
			&unit{atom: "a"},
			&unit{atom: "b"},
		},
	}
	actual := root.String()
	expect := `[set! a b]`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	actualSrc := root.DataSources()
	expectSrc := []string{"b"}
	if diff := cmp.Diff(expectSrc, actualSrc); diff != "" {
		t.Errorf("sources mismatch (-want +got):\n%s", diff)
	}
}
