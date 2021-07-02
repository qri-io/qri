package startf

import (
	"fmt"
	"reflect"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func analyzeScriptFile(thread *starlark.Thread, filename string) {
	err := doAnalyze(filename)
	if err != nil {
		fmt.Printf("analyzer error: %s\n", err)
	}
}

func doAnalyze(filename string) error {
	f, err := syntax.Parse(filename, nil, 0)
	if err != nil {
		return err
	}
	funcs, err := collectFunctionDefs(f.Stmts)
	if err != nil {
		return err
	}
	// Build a graph of all calls, Detect unused functions
	callGraph := buildCallGraph(funcs)

	fmt.Printf("----------------------------------------\n")
	showControlFlowForFunction(callGraph, "first_func")

	fmt.Printf("----------------------------------------\n")
	//showControlFlowForFunction(callGraph, "main_func")
	return nil
}

func collectFunctionDefs(stmts []syntax.Stmt) ([]*FuncResult, error) {
	functions := []*FuncResult{}
	for i, stmt := range stmts {
		switch item := stmt.(type) {
		case *syntax.DefStmt:
			res, err := analyzeFunction(item)
			if err != nil {
				return nil, err
			}
			functions = append(functions, res)
		default:
			fmt.Printf("%d: other top-level stmt\n", i)
		}
	}
	return functions, nil
}

func analyzeFunction(def *syntax.DefStmt) (*FuncResult, error) {
	params := make([]string, len(def.Params))
	for k, param := range def.Params {
		p := parameterName(param)
		params[k] = p
	}

	res, err := analyzeFuncBody(def.Body)
	if err != nil {
		return nil, err
	}
	res.name = def.Name.Name
	res.params = strings.Join(params, ",")
	res.body = def.Body

	return res, nil
}

func parameterName(e syntax.Expr) string {
	id, ok := e.(*syntax.Ident)
	if !ok {
		return fmt.Sprintf("<UNKNOWN: %s>", reflect.TypeOf(e))
	}
	return id.Name
}

type FuncResult struct {
	name   string
	params string
	calls  []string
	body   []syntax.Stmt
}

func NewFuncResult() *FuncResult {
	return &FuncResult{calls: []string{}}
}

func analyzeFuncBody(body []syntax.Stmt) (*FuncResult, error) {
	result := NewFuncResult()
	for k, stmt := range body {
		switch item := stmt.(type) {
		case *syntax.AssignStmt:
			// Is the rhs a function call
			calls := getFuncCallsInExpr(item.RHS)
			result.calls = append(result.calls, calls...)

		case *syntax.BranchStmt:
			// pass
			fmt.Printf("TODO func body %d: branch\n", k)
		case *syntax.DefStmt:
			// pass
			fmt.Printf("TODO func body %d: def\n", k)
		case *syntax.ExprStmt:
			// pass
			calls := getFuncCallsInExpr(item.X)
			result.calls = append(result.calls, calls...)

		case *syntax.ForStmt:
			// pass
			fmt.Printf("TODO func body %d: for\n", k)
		case *syntax.WhileStmt:
			// pass
			fmt.Printf("TODO func body %d: while\n", k)
		case *syntax.IfStmt:
			// pass
			fmt.Printf("TODO func body %d: if\n", k)
		case *syntax.LoadStmt:
			// pass
			fmt.Printf("TODO func body %d: load\n", k)
		case *syntax.ReturnStmt:
			// pass
			calls := getFuncCallsInExpr(item.Result)
			result.calls = append(result.calls, calls...)

		default:
			// pass
		}
	}
	return result, nil
}

func getFuncCallsInExpr(expr syntax.Expr) []string {
	switch item := expr.(type) {
	case *syntax.BinaryExpr:
		return append(getFuncCallsInExpr(item.X), getFuncCallsInExpr(item.Y)...)

	case *syntax.CallExpr:
		funcName := simpleExprToFuncName(item.Fn)
		result := make([]string, 0, 1 + len(item.Args))
		result = append(result, funcName)
		for _, arg := range item.Args {
			result = append(result, getFuncCallsInExpr(arg)...)
		}
		return result

	case *syntax.Comprehension:
		panic("not implemented")

	case *syntax.CondExpr:
		result := getFuncCallsInExpr(item.Cond)
		result = append(result, getFuncCallsInExpr(item.True)...)
		result = append(result, getFuncCallsInExpr(item.False)...)
		return result

	case *syntax.DictEntry:
		return append(getFuncCallsInExpr(item.Key), getFuncCallsInExpr(item.Value)...)

	case *syntax.DictExpr:
		result := make([]string, 0, len(item.List))
		for _, elem := range item.List {
			result = append(result, getFuncCallsInExpr(elem)...)
		}
		return result

	case *syntax.DotExpr:
		panic("not implemented")

	case *syntax.Ident:
		return []string{}

	case *syntax.IndexExpr:
		return append(getFuncCallsInExpr(item.X), getFuncCallsInExpr(item.Y)...)

	case *syntax.LambdaExpr:
		result := make([]string, 0, 1 + len(item.Params))
		result = append(result, getFuncCallsInExpr(item.Body)...)
		for _, elem := range item.Params {
			result = append(result, getFuncCallsInExpr(elem)...)
		}
		return result

	case *syntax.ListExpr:
		result := make([]string, 0, len(item.List))
		for _, elem := range item.List {
			result = append(result, getFuncCallsInExpr(elem)...)
		}
		return result

	case *syntax.Literal:
		return []string{}

	case *syntax.ParenExpr:
		return getFuncCallsInExpr(item.X)

	case *syntax.SliceExpr:
		result := getFuncCallsInExpr(item.X)
		result = append(result, getFuncCallsInExpr(item.Lo)...)
		result = append(result, getFuncCallsInExpr(item.Hi)...)
		result = append(result, getFuncCallsInExpr(item.Step)...)
		return result

	case *syntax.TupleExpr:
		result := make([]string, 0, len(item.List))
		for _, elem := range item.List {
			result = append(result, getFuncCallsInExpr(elem)...)
		}
		return result

	case *syntax.UnaryExpr:
		return getFuncCallsInExpr(item.X)

	}
	return nil
}

func simpleExprToFuncName(expr syntax.Expr) string {
	if item, ok := expr.(*syntax.Ident); ok {
		return item.Name
	}
	return fmt.Sprintf("<Unknown Name, Type: %q>", reflect.TypeOf(expr))
}

func showControlFlowForFunction(graph *CallGraph, fname string) {
	f := graph.lookup[fname]
	if f == nil {
		fmt.Printf("showing control flow, function %q not found", fname)
		return
	}
	body := f.fn.body

	controlFlow := newControlFlow()
	buildControlFlow(controlFlow, body)

	fmt.Printf("%s\n", controlFlow.stringify())
}

