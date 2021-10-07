package staticlark

import (
	"fmt"
	"reflect"
	"strings"

	//"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// build a list of functions
func collectFuncDefsTopLevelCalls(stmts []syntax.Stmt) ([]*FuncResult, []string, error) {
	functions := []*FuncResult{}
	topLevel := []string{}
	for _, stmt := range stmts {
		switch item := stmt.(type) {
		case *syntax.DefStmt:
			res, err := analyzeFunction(item)
			if err != nil {
				return nil, nil, err
			}
			functions = append(functions, res)
		default:
			calls := getFuncCallsInStmtList([]syntax.Stmt{stmt})
			topLevel = append(topLevel, calls...)
		}
	}
	return functions, topLevel, nil
}

// build a function object, contains calls to other functions
func analyzeFunction(def *syntax.DefStmt) (*FuncResult, error) {
	params := make([]string, len(def.Params))
	for k, param := range def.Params {
		p := parameterName(param)
		params[k] = p
	}

	res, err := buildFromFuncBody(def.Body)
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

// FuncResult is a function definition parsed from source code
type FuncResult struct {
	name   string
	params string
	calls  []string
	body   []syntax.Stmt
}

// NewFuncResult constructs a new FuncResult
func NewFuncResult() *FuncResult {
	return &FuncResult{calls: []string{}}
}

func buildFromFuncBody(body []syntax.Stmt) (*FuncResult, error) {
	result := NewFuncResult()
	for _, stmt := range body {
		switch item := stmt.(type) {
		case *syntax.AssignStmt:
			// Is the rhs a function call
			calls := getFuncCallsInExpr(item.RHS)
			result.calls = append(result.calls, calls...)

		case *syntax.BranchStmt:
			// TODO(dustmop)

		case *syntax.DefStmt:
			// TODO(dustmop)

		case *syntax.ExprStmt:
			calls := getFuncCallsInExpr(item.X)
			result.calls = append(result.calls, calls...)

		case *syntax.ForStmt:
			calls := getFuncCallsInExpr(item.X)
			calls = append(calls, getFuncCallsInStmtList(item.Body)...)
			result.calls = calls

		case *syntax.WhileStmt:
			calls := getFuncCallsInExpr(item.Cond)
			calls = append(calls, getFuncCallsInStmtList(item.Body)...)
			result.calls = calls

		case *syntax.IfStmt:
			calls := getFuncCallsInExpr(item.Cond)
			calls = append(calls, getFuncCallsInStmtList(item.True)...)
			calls = append(calls, getFuncCallsInStmtList(item.False)...)
			result.calls = calls

		case *syntax.LoadStmt:
			// pass

		case *syntax.ReturnStmt:
			calls := getFuncCallsInExpr(item.Result)
			result.calls = append(result.calls, calls...)

		default:
			// pass
		}
	}
	return result, nil
}

func getFuncCallsInStmtList(listStmt []syntax.Stmt) []string {
	result := make([]string, 0)

	for _, stmt := range listStmt {
		switch item := stmt.(type) {
		case *syntax.AssignStmt:
			calls := getFuncCallsInExpr(item.LHS)
			calls = append(calls, getFuncCallsInExpr(item.RHS)...)
			result = append(result, calls...)

		case *syntax.BranchStmt:
			// TODO(dustmop)

		case *syntax.DefStmt:
			// TODO(dustmop)

		case *syntax.ExprStmt:
			calls := getFuncCallsInExpr(item.X)
			result = append(result, calls...)

		case *syntax.ForStmt:
			// TODO(dustmop)

		case *syntax.WhileStmt:
			// TODO(dustmop)

		case *syntax.IfStmt:
			calls := getFuncCallsInExpr(item.Cond)
			calls = append(calls, getFuncCallsInStmtList(item.True)...)
			calls = append(calls, getFuncCallsInStmtList(item.False)...)
			result = append(result, calls...)

		case *syntax.LoadStmt:
			// TODO(dustmop)

		case *syntax.ReturnStmt:
			// TODO(dustmop)

		}
	}

	return result
}

func getFuncCallsInExpr(expr syntax.Expr) []string {
	switch item := expr.(type) {
	case *syntax.BinaryExpr:
		return append(getFuncCallsInExpr(item.X), getFuncCallsInExpr(item.Y)...)

	case *syntax.CallExpr:
		funcName := simpleExprToFuncName(item.Fn)
		result := make([]string, 0, 1+len(item.Args))
		result = append(result, funcName)
		for _, arg := range item.Args {
			result = append(result, getFuncCallsInExpr(arg)...)
		}
		return result

	case *syntax.Comprehension:
		// TODO
		return []string{}

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
		// TODO
		return []string{}

	case *syntax.Ident:
		return []string{}

	case *syntax.IndexExpr:
		return append(getFuncCallsInExpr(item.X), getFuncCallsInExpr(item.Y)...)

	case *syntax.LambdaExpr:
		result := make([]string, 0, 1+len(item.Params))
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
