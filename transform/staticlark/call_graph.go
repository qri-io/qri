package staticlark

import (
	"fmt"
	"sort"
	"strings"
)

// CallGraph is a graph of function nodes and what they call
type CallGraph struct {
	nodes  []*FuncNode
	lookup map[string]*FuncNode
}

// FuncNode is a single function definition and body, along with
// additional information derived from static analysis
type FuncNode struct {
	name   string
	fn     *FuncResult
	outs   []*FuncNode
	reach  bool
	height int
}

func buildCallGraph(functions []*FuncResult, entryPoints []string) *CallGraph {
	// Add some built-in functions to the symbol table
	symtable := map[string]*FuncResult{}
	for _, f := range functions {
		symtable[f.name] = f
	}
	symtable["print"] = &FuncResult{name: "print"}
	symtable["range"] = &FuncResult{name: "range"}

	// Build the call graph
	graph := &CallGraph{
		nodes:  make([]*FuncNode, 0, len(functions)),
		lookup: make(map[string]*FuncNode),
	}
	for _, f := range functions {
		addToCallGraph(f, graph, symtable)
	}

	for _, n := range graph.nodes {
		addCallHeight(n)
	}

	// Determine reachability using the given entry points
	if entryPoints != nil {
		for _, entry := range entryPoints {
			root := graph.lookup[entry]
			if root != nil {
				markReachable(root)
			}
		}
	}

	return graph
}

func addToCallGraph(f *FuncResult, graph *CallGraph, symtable map[string]*FuncResult) *FuncNode {
	me, ok := graph.lookup[f.name]
	if ok {
		return me
	}
	me = &FuncNode{
		name: f.name,
		fn:   f,
		outs: make([]*FuncNode, 0),
	}
	for _, call := range f.calls {
		child, ok := symtable[call]
		if !ok {
			// Function not found, ignore it
			continue
		}
		n := addToCallGraph(child, graph, symtable)
		me.outs = append(me.outs, n)
	}
	graph.lookup[f.name] = me
	graph.nodes = append(graph.nodes, me)
	return me
}

func addCallHeight(node *FuncNode) {
	maxChild := -1
	for _, fn := range node.outs {
		addCallHeight(fn)
		if fn.height > maxChild {
			maxChild = fn.height
		}
	}
	node.height = maxChild + 1
}

func markReachable(node *FuncNode) {
	node.reach = true
	for _, call := range node.outs {
		markReachable(call)
	}
}

func (cg *CallGraph) findUnusedFuncs() []string {
	// Recursively walk the tree to find unreachable nodes
	unusedNames := map[string]struct{}{}
	for _, f := range cg.nodes {
		checkFuncNodeUnused(f, unusedNames)
	}
	// Sort the function names
	results := make([]string, 0, len(unusedNames))
	for fname := range unusedNames {
		results = append(results, fname)
	}
	sort.Strings(results)
	return results
}

func checkFuncNodeUnused(node *FuncNode, unusedNames map[string]struct{}) {
	if !node.reach {
		unusedNames[node.name] = struct{}{}
	}
	for _, call := range node.outs {
		checkFuncNodeUnused(call, unusedNames)
	}
}

// String creates a string representation of functions in the call graph
func (cg *CallGraph) String() string {
	text := ""
	for _, n := range cg.nodes {
		text += stringifyNode(n, 0)
	}
	return text
}

func stringifyNode(n *FuncNode, depth int) string {
	padding := strings.Repeat(" ", depth)
	seen := map[string]struct{}{}
	text := fmt.Sprintf("%s%s\n", padding, n.name)
	for _, call := range n.outs {
		if _, ok := seen[call.name]; ok {
			continue
		}
		seen[call.name] = struct{}{}
		text += stringifyNode(call, depth+1)
	}
	return text
}
