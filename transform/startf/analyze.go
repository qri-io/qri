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

	text := string(data)
	fmt.Printf("%s\n================================\n\n", text)

	for i, stmt := range f.Stmts {
		switch item := stmt.(type) {
		case *syntax.DefStmt:
			err := analyzeFunction(item)
			if err != nil {
				return err
			}
		default:
			fmt.Printf("%d: other top-level stmt\n", i)
		}
	}

	return nil
}

func analyzeFunction(def *syntax.DefStmt) error {
	fmt.Printf("def func: %q\n", def.Name.Name)

	numParams := len(def.Params)
	_ = numParams
	params := make([]string, numParams)
	for k, param := range def.Params {
		p := parameterName(param)
		params[k] = p
	}
	fmt.Printf(" params: (%s)\n", strings.Join(params, ","))

	err := analyzeFuncBody(def.Body)
	if err != nil {
		return err
	}

	fmt.Printf("\n")

	return nil
}

func parameterName(e syntax.Expr) string {
	id, ok := e.(*syntax.Ident)
	if !ok {
		return fmt.Sprintf("<UNKNOWN: %s>", reflect.TypeOf(e))
	}
	return id.Name
}

func analyzeFuncBody(body []syntax.Stmt) error {
	for k, stmt := range body {
		data, err := json.Marshal(stmt)
		if err != nil {
			return err
		}
		text := string(data)
		//fmt.Printf("%d: %s\n", k, text)
		_ = text

		switch item := stmt.(type) {
		case *syntax.AssignStmt:
			// pass
			fmt.Printf("%d: assign\n", k)
			_ = item
		case *syntax.BranchStmt:
			// pass
			fmt.Printf("%d: branch\n", k)
		case *syntax.DefStmt:
			// pass
			fmt.Printf("%d: def\n", k)
		case *syntax.ExprStmt:
			// pass
			fmt.Printf("%d: expr\n", k)
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
	return nil
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
