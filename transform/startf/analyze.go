package startf

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func analyzeScriptFile(thread *starlark.Thread, filename string) {
	err := doAnalyze(filename)
	if err != nil {
		panic(err)
	}
}

func doAnalyze(filename string) error {
	// ExecFile(thread *Thread, filename string, src interface{}, predeclared StringDict)
	// SourceProgram(filename string, src interface{}, isPredeclared func(string) bool)
	// f, err := syntax.Parse(filename string, src interface{}, 0 ?)
	fmt.Printf("analyze: %s\n", filename)

	f, err := syntax.Parse(filename, nil, 0)
	if err != nil {
		return err
	}

	fmt.Printf("Parsed successfully!\n")
	data, err := json.MarshalIndent(f, "", " ")
	if err != nil {
		return err
	}

	functions := []*FuncResult{}

	text := string(data)
	fmt.Printf("%s\n================================\n\n", text)

	for i, stmt := range f.Stmts {
		switch item := stmt.(type) {
		case *syntax.DefStmt:
			res, err := analyzeFunction(item)
			if err != nil {
				return err
			}
			functions = append(functions, res)
		default:
			fmt.Printf("%d: other top-level stmt\n", i)
		}
	}

	fmt.Printf("----------------------------------------\n")
	for _, f := range functions {
		fmt.Printf("def %s(%s)\n", f.name, f.params)
		for _, c := range f.calls {
			fmt.Printf(" %s()\n", c)
		}
		fmt.Printf("\n")
	}

	return nil
}

func analyzeFunction(def *syntax.DefStmt) (*FuncResult, error) {
	fmt.Printf("def func: %q\n", def.Name.Name)

	numParams := len(def.Params)
	_ = numParams
	params := make([]string, numParams)
	for k, param := range def.Params {
		p := parameterName(param)
		params[k] = p
	}
	fmt.Printf(" params: (%s)\n", strings.Join(params, ","))

	res, err := analyzeFuncBody(def.Body)
	if err != nil {
		return nil, err
	}
	res.name = def.Name.Name
	res.params = strings.Join(params, ",")

	fmt.Printf("\n")

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
}

func NewFuncResult() *FuncResult {
	return &FuncResult{calls: []string{}}
}

func analyzeFuncBody(body []syntax.Stmt) (*FuncResult, error) {
	result := NewFuncResult()
	for k, stmt := range body {
		data, err := json.Marshal(stmt)
		if err != nil {
			return nil, err
		}
		text := string(data)
		//fmt.Printf("%d: %s\n", k, text)
		_ = text

		switch item := stmt.(type) {
		case *syntax.AssignStmt:
			// pass
			fmt.Printf("%d: assign\n", k)
			_ = item
			// Is the rhs a function call?

		case *syntax.BranchStmt:
			// pass
			fmt.Printf("%d: branch\n", k)
		case *syntax.DefStmt:
			// pass
			fmt.Printf("%d: def\n", k)
		case *syntax.ExprStmt:
			// pass
			fmt.Printf("%d: expr\n", k)
			// Is this a function call? (almost *certainly* it is)
			calls := getFuncCallsInExpr(item.X)
			result.calls = append(result.calls, calls...)

		case *syntax.ForStmt:
			// pass
			fmt.Printf("%d: for\n", k)
		case *syntax.WhileStmt:
			// pass
			fmt.Printf("%d: while\n", k)
		case *syntax.IfStmt:
			// pass
			fmt.Printf("%d: if\n", k)
		case *syntax.LoadStmt:
			// pass
			fmt.Printf("%d: load\n", k)
		case *syntax.ReturnStmt:
			// pass
			fmt.Printf("%d: return\n", k)
		default:
			// pass
		}
	}
	return result, nil
}

// Stmt:
//  AssignStmt step
//  BranchStmt control-flow -> jump (BREAK | CONTINUE | PASS)
//  DefStmt    ?
//  ExprStmt   step
//  ForStmt    control-flow -> loop
//  WhileStmt  control-flow -> loop
//  IfStmt     control-flow -> branch
//  LoadStmt   ?
//  ReturnStmt control-flow -> termination

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
		// I think that this is correct?
		fmt.Printf("Ident is not a FuncCall I think?\n")
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

/*
	switch item := expr.(type) {
	case *syntax.BinaryExpr:
		// pass
	case *syntax.CallExpr:
		// pass
	case *syntax.ComprehensionExpr:
		// pass
	case *syntax.CondExpr:
		// pass
	case *syntax.DictEntry:
		// pass
	case *syntax.DictExpr:
		// pass
	case *syntax.DotExpr:
		// pass
	case *syntax.Ident:
		return item.Name
	case *syntax.IndexExpr:

	case *syntax.LambdaExpr:

	case *syntax.ListExpr:

	case *syntax.Literal:

	case *syntax.ParenExpr:

	case *syntax.SliceExpr:

	case *syntax.TupleExpr:

	case *syntax.UnaryExpr:

	}
}
*/
