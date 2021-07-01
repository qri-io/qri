package startf

import (
	//"encoding/json"
	"fmt"
	"go.starlark.net/syntax"
	"strconv"
	"strings"
)

type ControlFlow struct {
	Nodes []*CodeBlock
	Curr  *CodeBlock
}

func newControlFlow() *ControlFlow {
	return &ControlFlow{}
}

func (c *ControlFlow) display() {
	for i, n := range c.Nodes {
		fmt.Printf("---------------------\n")
		fmt.Printf("%d:\n", i)
		fmt.Printf("%s\n", n.Code)
		fmt.Printf("out: %v\n", n.Outs)
	}
}

func (c *ControlFlow) stringify() string {
	result := ""
	for i, n := range c.Nodes {
		result += fmt.Sprintf("%d: ", i)
		for j, c := range n.Code {
			if j == 0 {
				result += fmt.Sprintf("%s\n", c)
			} else {
				padding := strings.Repeat(" ", 3)
				result += fmt.Sprintf("%s%s\n", padding, c)
			}
		}
		if len(n.Code) == 0 {
			result += "-\n"
		}
		outPaths := make([]string, len(n.Outs))
		for j, out := range n.Outs {
			outPaths[j] = strconv.Itoa(out)
		}
		result += fmt.Sprintf("  out: %s\n", strings.Join(outPaths, ","))
	}
	return result
}

func (c *ControlFlow) prepare() {
	if c.Nodes == nil {
		c.Nodes = append(c.Nodes, newCodeBlock())
		c.Curr = c.Nodes[len(c.Nodes)-1]
	}
}

func (c *ControlFlow) add(line string) {
	c.prepare()
	c.Curr.Code = append(c.Curr.Code, line)
}

func (c *ControlFlow) makeNew() int {
	c.prepare()

	nextIndex := len(c.Nodes)
	c.Curr.Outs = append(c.Curr.Outs, nextIndex)

	c.Nodes = append(c.Nodes, newCodeBlock())
	c.Curr = c.Nodes[len(c.Nodes)-1]

	return c.get()
}

func (c *ControlFlow) makeNewNoArrow() int {
	c.prepare()

	c.Nodes = append(c.Nodes, newCodeBlock())
	c.Curr = c.Nodes[len(c.Nodes)-1]

	return c.get()
}

func (c *ControlFlow) get() int {
	return len(c.Nodes) - 1
}

func (c *ControlFlow) poke(index, value int) {
	c.Nodes[index].Outs = append(c.Nodes[index].Outs, value)
}

func (c *ControlFlow) concat(other *ControlFlow) int {
	size := len(c.Nodes)

	for _, n := range other.Nodes {
		replace := make([]int, 0)
		for _, out := range n.Outs {
			replace = append(replace, out + size)
		}
		n.Outs = replace
	}

	c.Nodes = append(c.Nodes, other.Nodes...)
	c.Curr = c.Nodes[len(c.Nodes)-1]
	return size
}

type CodeBlock struct {
	Code []string
	Outs []int
}

func newCodeBlock() *CodeBlock {
	return &CodeBlock{
		Code: []string{},
		Outs: []int{},
	}
}

func buildControlFlow(control *ControlFlow, stmtList []syntax.Stmt) {
	for _, line := range stmtList {
		buildControlFlowSingleNode(control, line)
	}
}

func buildControlFlowSingleNode(control *ControlFlow, stmt syntax.Stmt) {

	switch item := stmt.(type) {
	case *syntax.AssignStmt:
		// TODO: Also record vars in LHS and RHS
		assignLine := assignmentToText(item)
		control.add(assignLine)

	case *syntax.BranchStmt:
		fmt.Printf("~~~ TODO: branch stmt\n")

	case *syntax.DefStmt:
		fmt.Printf("~~~ TODO: def stmt\n")

	case *syntax.ExprStmt:
		// TODO: Also record vars in Params
		funcCallLine := funcCallToText(item)
		control.add(funcCallLine)

	case *syntax.ForStmt:
		// Add new block, connect old one here
		control.makeNew()

		condLine := condToText(item.X)
		control.add(condLine)

		loopIndex := control.get()

		// TODO: item.Vars, item.Expr

		loopBody := newControlFlow()
		buildControlFlow(loopBody, item.Body)
		bodyIndex := control.concat(loopBody)

		lastIndex := control.get()
		afterIndex := control.makeNewNoArrow()

		control.poke(loopIndex, bodyIndex)
		control.poke(loopIndex, afterIndex)
		control.poke(lastIndex, loopIndex)

	case *syntax.WhileStmt:
		fmt.Printf("~~~ TODO: while stmt\n")

	case *syntax.IfStmt:
		control.makeNew()
		// Condition of If statement
		condLine := condToText(item.Cond)
		control.add(condLine)
		c := control.get()

		ifTrueBranch := newControlFlow()
		buildControlFlow(ifTrueBranch, item.True)
		t := control.concat(ifTrueBranch)

		// TODO: Handle false being empty (no `else`)

		ifFalseBranch := newControlFlow()
		buildControlFlow(ifFalseBranch, item.False)
		f := control.concat(ifFalseBranch)

		u := control.makeNewNoArrow()

		control.poke(c, t)
		control.poke(c, f)
		control.poke(t, u)
		control.poke(f, u)

	case *syntax.LoadStmt:
		fmt.Printf("~~~ TODO: load stmt\n")

	case *syntax.ReturnStmt:
		fmt.Printf("~~~ TODO: return stmt\n")

	}
}

func assignmentToText(assign *syntax.AssignStmt) string {
	result := "set! "
	if ident, ok := assign.LHS.(*syntax.Ident); ok {
		result = result + ident.Name
	} else {
		result = result + "???"
	}
	result = result + " = "
	if ident, ok := assign.RHS.(*syntax.Ident); ok {
		result = result + ident.Name
	} else if val, ok := assign.RHS.(*syntax.Literal); ok {
		result = result + val.Raw
	} else {
		result = result + "???"
	}
	return result
}

func funcCallToText(expr *syntax.ExprStmt) string {
	e := expr.X
	switch item := e.(type) {
	case *syntax.BinaryExpr:
		return "binary()"

	case *syntax.CallExpr:
		fn := item.Fn
		funcCallIdent := fn.(*syntax.Ident)
		return fmt.Sprintf("%s()", funcCallIdent.Name)

	case *syntax.Comprehension:
		return "comp()"

	case *syntax.CondExpr:
		return "cond()"

	case *syntax.DictEntry:
		return "dictEntry()"

	case *syntax.DictExpr:
		return "dict()"

	case *syntax.DotExpr:
		return "dot()"

	case *syntax.Ident:
		return fmt.Sprintf("%s()", item.Name)

	case *syntax.IndexExpr:
		return "index()"

	case *syntax.LambdaExpr:
		return "lambda()"

	case *syntax.ListExpr:
		return "list()"

	case *syntax.Literal:
		return "literal()"

	case *syntax.ParenExpr:
		return "paren()"

	case *syntax.SliceExpr:
		return "slice()"

	case *syntax.TupleExpr:
		return "tuple()"

	case *syntax.UnaryExpr:
		return "unary()"

	}
	return "????()"
}

func condToText(expr syntax.Expr) string {
	switch item := expr.(type) {
	case *syntax.BinaryExpr:
		_ = item
		return "if binary()"

	case *syntax.CallExpr:
		return "if call()"

	case *syntax.Comprehension:
		return "if comprehension()"

	case *syntax.CondExpr:
		return "if cond()"

	case *syntax.DictEntry:
		return "if dict-entry()"

	case *syntax.DictExpr:
		return "if dict()"

	case *syntax.DotExpr:
		return "if dot()"

	case *syntax.Ident:
		return "if ident()"

	case *syntax.IndexExpr:
		return "if index()"

	case *syntax.LambdaExpr:
		return "if lambda()"

	case *syntax.ListExpr:
		return "if list()"

	case *syntax.Literal:
		return "if literal()"

	case *syntax.ParenExpr:
		return "if paren()"

	case *syntax.SliceExpr:
		return "if slice()"

	case *syntax.TupleExpr:
		return "if tuple()"

	case *syntax.UnaryExpr:
		return "if unary()"

	}

	return "if ????"
}
