package staticlark

import (
	"fmt"
	"strconv"
	"strings"

	"go.starlark.net/syntax"
)

// controlFlow represents the control flow within a single function
// It is a list of blocks, where each block is one or more code units
// that have linear control flow. In addition, each block has a list
// of outgoing edges that reference which block control flow may move
// to next. These edges are represented by indexes into the control
// flow graph's block list.
//
// Example:
//
// The code:
//
//   a = 1
//   if a > b:
//     print('big')
//   print('done')
//
// The graph represented abstractly:
//
// +-------+
// | a = 1 |--+
// +-------+  |
//            |
//   +--------+
//   |
//   v
// +----------+
// | if a > b |------+
// +----------+      |
//                   |
//   +---------------+
//   |               |
//   v               |
// +--------------+  |
// | print('big') |--+
// +--------------+  |
//                   |
//   +---------------+
//   |
//   v
// +---------------+
// | print('done') |
// +---------------+
//
// Control flow, stringified:
//
// 0: [set! a 1]
//   outs: 1
// 1: [if [> a b]]
//   outs: 2,3
// 2: [print 'big']
//   outs: 3
// 3: [print 'done']
//   outs: -
type controlFlow struct {
	blocks []*codeBlock
}

func newControlFlow() *controlFlow {
	return &controlFlow{}
}

func newControlFlowFromFunc(fn *funcNode) (*controlFlow, error) {
	builder := newControlFlowBuilder()
	cf := builder.build(fn.body)
	return cf, nil
}

func (c *controlFlow) stringify() string {
	result := ""
	for i, block := range c.blocks {
		result += fmt.Sprintf("%d: ", i)
		for j, unit := range block.units {
			if j == 0 {
				result += fmt.Sprintf("%s\n", unit)
			} else {
				padding := strings.Repeat(" ", 3)
				result += fmt.Sprintf("%s%s\n", padding, unit)
			}
		}
		if len(block.units) == 0 {
			result += "-\n"
		}
		if len(block.edges) == 0 {
			result += "  out: -\n"
		} else if len(block.edges) == 1 && block.edges[0] == -1 {
			result += "  out: return\n"
		} else if len(block.edges) == 1 && block.edges[0] == -2 {
			result += "  out: break\n"
		} else {
			outs := make([]string, len(block.edges))
			for j, edge := range block.edges {
				outs[j] = strconv.Itoa(edge)
			}
			result += fmt.Sprintf("  out: %s\n", strings.Join(outs, ","))
		}
	}
	return result
}

// codeBlock is a single block of linear control flow, along with
// a list of outgoing edges, represented as indexes into other
// blocks in the control flow
type codeBlock struct {
	units []*unit
	edges []int
}

func newCodeBlock() *codeBlock {
	return &codeBlock{
		units: []*unit{},
	}
}

// cfBuilder is a builder that creates a control flow
type cfBuilder struct {
	// the entire control flow being built
	flow *controlFlow
	// current block being added to
	curr *codeBlock
	// references to blocks that do not have outgoing edges
	dangling []int
}

func newControlFlowBuilder() *cfBuilder {
	builder := &cfBuilder{}
	return builder
}

// create a new block if there is no current block being built
func (builder *cfBuilder) ensureBlock() {
	if builder.flow.blocks == nil || builder.curr == nil {
		builder.makeBlock()
	}
}

// put a unit at the end of the current block
func (builder *cfBuilder) put(unit *unit) {
	builder.curr.units = append(builder.curr.units, unit)
}

// for each referenced block, add `dest` to the outgoing edges
func (builder *cfBuilder) addEdges(blockRefs []int, dest []int) {
	for _, ref := range blockRefs {
		builder.flow.blocks[ref].edges = append(builder.flow.blocks[ref].edges, dest...)
	}
}

// add a new block to the control flow, and point any dangling edges to it
func (builder *cfBuilder) makeBlock() {
	if len(builder.dangling) != 0 {
		index := builder.refNext()
		builder.addEdges(builder.dangling, []int{index})
	}

	index := builder.refNext()
	builder.flow.blocks = append(builder.flow.blocks, newCodeBlock())
	builder.curr = builder.flow.blocks[index]
	builder.dangling = []int{index}
}

// mark the current block as done, return refs to blocks with dangling edges
func (builder *cfBuilder) finish() []int {
	refs := builder.dangling
	builder.curr = nil
	builder.dangling = nil
	return refs
}

// get reference to the next block, as an index
func (builder *cfBuilder) refNext() int {
	return len(builder.flow.blocks)
}

// build is the main entry point for the builder. Takes a body of a
// function and creates a control flow for that body
func (builder *cfBuilder) build(stmtList []syntax.Stmt) *controlFlow {
	builder.flow = newControlFlow()
	builder.buildSubGraph(stmtList)
	return builder.flow
}

// build a sub graph of an already in-use builder, return the dangling
// edges at the end of that sub-graph
func (builder *cfBuilder) buildSubGraph(stmtList []syntax.Stmt) []int {
	for _, line := range stmtList {
		builder.buildSingleNode(line)
	}
	return builder.finish()
}

func (builder *cfBuilder) buildSingleNode(stmt syntax.Stmt) {
	switch item := stmt.(type) {
	case *syntax.AssignStmt:
		assignUnit := assignmentToUnit(item)
		builder.ensureBlock()
		builder.put(assignUnit)

	case *syntax.BranchStmt:
		// TODO(dustmop): support other operations, like `continue`
		builder.ensureBlock()
		builder.put(&unit{atom: "[break]"})
		// TODO(dustmop): have the builder track the inner-most loop
		// being built, have this edge point to the end of that loop
		builder.addEdges(builder.dangling, []int{-2})
		builder.dangling = nil

	case *syntax.DefStmt:
		// TODO(dustmop): inner functions need to be supported

	case *syntax.ExprStmt:
		builder.ensureBlock()
		builder.put(exprStatementToUnit(item))

	case *syntax.ForStmt:
		startPos := builder.refNext()
		builder.makeBlock()

		// condition for loop
		checkUnit := &unit{atom: "for"}
		checkUnit.tail = []*unit{exprToUnit(item.Vars), exprToUnit(item.X)}
		builder.put(checkUnit)
		loopEntry := builder.finish()

		bodyPos := builder.refNext()
		loopLeave := builder.buildSubGraph(item.Body)

		// add edges to create the loop flow
		builder.addEdges(loopEntry, []int{bodyPos})
		builder.addEdges(loopLeave, []int{startPos})
		builder.dangling = []int{startPos}

	case *syntax.WhileStmt:
		// NOTE: analyzer does not support while loops

	case *syntax.IfStmt:
		builder.makeBlock()

		// condition of the if
		condLine := &unit{atom: "if"}
		condLine.tail = append(condLine.tail, exprToUnit(item.Cond))
		builder.put(condLine)
		branchEntry := builder.finish()

		truePos := builder.refNext()
		exitTrue := builder.buildSubGraph(item.True)

		falsePos := builder.refNext()
		exitFalse := builder.buildSubGraph(item.False)

		if len(exitFalse) > 0 {
			builder.addEdges(branchEntry, []int{truePos, falsePos})
			builder.dangling = append(exitTrue, exitFalse...)
		} else {
			builder.addEdges(branchEntry, []int{truePos})
			builder.dangling = append(exitTrue, branchEntry...)
		}

	case *syntax.LoadStmt:
		// nothing to do

	case *syntax.ReturnStmt:
		builder.ensureBlock()
		retLine := &unit{atom: "return"}
		retLine.tail = []*unit{exprToUnit(item.Result)}
		builder.put(retLine)
		builder.addEdges(builder.dangling, []int{-1})
		builder.dangling = nil

	}
}

func assignmentToUnit(assign *syntax.AssignStmt) *unit {
	// left hand side of assignment
	lhs := ""
	if ident, ok := assign.LHS.(*syntax.Ident); ok {
		lhs = ident.Name
	} else {
		lhs = fmt.Sprintf("TODO:%T", assign.LHS)
	}

	// right hand side of assignment
	rhs := &unit{}
	if ident, ok := assign.RHS.(*syntax.Ident); ok {
		rhs.Push(ident.Name)
	} else if val, ok := assign.RHS.(*syntax.Literal); ok {
		rhs.Push(val.Raw)
	} else if binExp, ok := assign.RHS.(*syntax.BinaryExpr); ok {
		tree := binaryOpToUnit(binExp)
		rhs.tail = append(rhs.tail, tree)
	} else if _, ok := assign.RHS.(*syntax.CallExpr); ok {
		unit := exprToUnit(assign.RHS)
		rhs.tail = append(rhs.tail, unit)
	} else {
		rhs.Push(fmt.Sprintf("TODO:%T", assign.RHS))
	}

	result := &unit{atom: "set!", where: getWhere(assign)}
	result.Push(lhs)
	if assign.Op == syntax.EQ {
		result.tail = append(result.tail, rhs.tail...)
	} else {
		result.tail = buildAssignOp(syntaxOpToString(assign.Op), lhs, rhs.tail)
	}
	return result
}

func syntaxOpToString(op syntax.Token) string {
	switch op {
	case syntax.EQ:
		return "="
	case syntax.PLUS_EQ:
		return "+="
	case syntax.MINUS_EQ:
		return "-="
	case syntax.STAR_EQ:
		return "*="
	case syntax.SLASH_EQ:
		return "/="
	}
	return "?"
}

func buildAssignOp(opText, ident string, expr []*unit) []*unit {
	prev := []*unit{&unit{atom: ident}}
	return append(prev, &unit{atom: opText, tail: append([]*unit{&unit{atom: ident}}, expr...)})
}

func exprStatementToUnit(expr *syntax.ExprStmt) *unit {
	e := expr.X
	switch item := e.(type) {
	case *syntax.BinaryExpr:
		return toUnitTODO("binary()")

	case *syntax.CallExpr:
		fn := item.Fn
		callName := simpleExprToFuncName(fn)
		tail := []*unit{}
		for _, e := range item.Args {
			// TODO: exprToUnit(e).String() shouldn't collapse to string
			tail = append(tail, &unit{atom: exprToUnit(e).String()})
		}
		return &unit{atom: callName, tail: tail, where: getWhere(expr)}

	case *syntax.Comprehension:
		return toUnitTODO("comp()")

	case *syntax.CondExpr:
		return toUnitTODO("cond()")

	case *syntax.DictEntry:
		return toUnitTODO("dictEntry()")

	case *syntax.DictExpr:
		return toUnitTODO("dict()")

	case *syntax.DotExpr:
		return toUnitTODO("dot()")

	case *syntax.Ident:
		return toUnitTODO("%s()")

	case *syntax.IndexExpr:
		return toUnitTODO("index()")

	case *syntax.LambdaExpr:
		return toUnitTODO("lambda()")

	case *syntax.ListExpr:
		return toUnitTODO("list()")

	case *syntax.Literal:
		return toUnitTODO("literal()")

	case *syntax.ParenExpr:
		return toUnitTODO("paren()")

	case *syntax.SliceExpr:
		return toUnitTODO("slice()")

	case *syntax.TupleExpr:
		return toUnitTODO("tuple()")

	case *syntax.UnaryExpr:
		return toUnitTODO("unary()")

	}
	return toUnitTODO("????()")
}

func binaryOpToUnit(binExp *syntax.BinaryExpr) *unit {
	res := &unit{}
	res.atom = binExp.Op.String()
	res.tail = []*unit{exprToUnit(binExp.X), exprToUnit(binExp.Y)}
	return res
}

func exprToUnit(expr syntax.Expr) *unit {
	switch item := expr.(type) {
	case *syntax.BinaryExpr:
		return binaryOpToUnit(item)

	case *syntax.CallExpr:
		fn := item.Fn
		callName := simpleExprToFuncName(fn)
		tail := []*unit{}
		for _, e := range item.Args {
			// TODO: exprToUnit(e).String() shouldn't collapse to string
			tail = append(tail, &unit{atom: exprToUnit(e).String()})
		}
		return &unit{atom: callName, tail: tail, where: getWhere(expr)}

	case *syntax.Comprehension:
		return toUnitTODO("{comprehension}")

	case *syntax.CondExpr:
		return toUnitTODO("{condExpr}")

	case *syntax.DictEntry:
		return toUnitTODO("{dictEntry}")

	case *syntax.DictExpr:
		return toUnitTODO("{dictExpr}")

	case *syntax.DotExpr:
		return toUnitTODO("{dotExpr}")

	case *syntax.Ident:
		return &unit{atom: item.Name}

	case *syntax.IndexExpr:
		return toUnitTODO("{indexExpr}")

	case *syntax.LambdaExpr:
		return toUnitTODO("{lambdaExpr}")

	case *syntax.ListExpr:
		return toUnitTODO("{listExpr}")

	case *syntax.Literal:
		return &unit{atom: item.Raw}

	case *syntax.ParenExpr:
		return toUnitTODO("{parenExpr}")

	case *syntax.SliceExpr:
		return toUnitTODO("{sliceExpr}")

	case *syntax.TupleExpr:
		return toUnitTODO("{tupleExpr}")

	case *syntax.UnaryExpr:
		return toUnitTODO("{unaryExpr}")

	default:
		return toUnitTODO("{unknown}")
	}
}

func getWhere(n syntax.Node) syntax.Position {
	start, _ := n.Span()
	return start
}
