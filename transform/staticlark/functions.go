package staticlark

import (
	"fmt"
	"reflect"
	"strings"

	"go.starlark.net/syntax"
)

// build a list of functions
func collectFuncDefsTopLevelCalls(stmts []syntax.Stmt) ([]*funcResult, []string, error) {
	functions := []*funcResult{}
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
func analyzeFunction(def *syntax.DefStmt) (*funcResult, error) {
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

// funcResult is a function definition parsed from source code
type funcResult struct {
	name   string
	params string
	calls  []string
	body   []syntax.Stmt
}

// newFuncResult constructs a new funcResult
func newFuncResult() *funcResult {
	return &funcResult{calls: []string{}}
}

func buildFromFuncBody(body []syntax.Stmt) (*funcResult, error) {
	result := newFuncResult()
	result.calls = getFuncCallsInStmtList(body)
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
			// pass

		case *syntax.DefStmt:
			// TODO(dustmop): Add this definition to the lexical scope

		case *syntax.ExprStmt:
			calls := getFuncCallsInExpr(item.X)
			result = append(result, calls...)

		case *syntax.ForStmt:
			calls := getFuncCallsInExpr(item.X)
			calls = append(calls, getFuncCallsInStmtList(item.Body)...)
			result = append(result, calls...)

		case *syntax.WhileStmt:
			calls := getFuncCallsInExpr(item.Cond)
			calls = append(calls, getFuncCallsInStmtList(item.Body)...)
			result = append(result, calls...)

		case *syntax.IfStmt:
			calls := getFuncCallsInExpr(item.Cond)
			calls = append(calls, getFuncCallsInStmtList(item.True)...)
			calls = append(calls, getFuncCallsInStmtList(item.False)...)
			result = append(result, calls...)

		case *syntax.LoadStmt:
			// pass

		case *syntax.ReturnStmt:
			calls := getFuncCallsInExpr(item.Result)
			result = append(result, calls...)

		}
	}

	return result
}

func getFuncCallsInExpr(expr syntax.Expr) []string {
	if expr == nil {
		return []string{}
	}
	switch item := expr.(type) {
	case *syntax.BinaryExpr:
		return append(getFuncCallsInExpr(item.X), getFuncCallsInExpr(item.Y)...)

	case *syntax.CallExpr:
		// TODO(dustmop): Add lexical scoping so that inner functions are
		// correctly associated with their call sites
		funcName := simpleExprToFuncName(item.Fn)
		result := make([]string, 0, 1+len(item.Args))
		result = append(result, funcName)
		for _, arg := range item.Args {
			result = append(result, getFuncCallsInExpr(arg)...)
		}
		return result

	case *syntax.Comprehension:
		result := getFuncCallsInExpr(item.Body)
		return result

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
	if item, ok := expr.(*syntax.DotExpr); ok {
		lhs := simpleExprToFuncName(item.X)
		return fmt.Sprintf("%s.%s", lhs, item.Name.Name)
	}
	return fmt.Sprintf("<Unknown Name, Type: %q>", reflect.TypeOf(expr))
}
