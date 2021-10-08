package staticlark

import (
	"fmt"
	"sort"
	"strings"
)

// callGraph is a graph of function nodes and what they call
type callGraph struct {
	nodes  []*funcNode
	lookup map[string]*funcNode
}

// funcNode is a single function definition and body, along with
// additional information derived from static analysis
type funcNode struct {
	name   string
	fn     *funcResult
	outs   []*funcNode
	reach  bool
	height int
}

func buildCallGraph(functions []*funcResult, entryPoints []string) *callGraph {
	// Add some built-in functions to the symbol table
	symtable := map[string]*funcResult{}
	for _, f := range functions {
		symtable[f.name] = f
	}
	symtable["print"] = &funcResult{name: "print"}
	symtable["range"] = &funcResult{name: "range"}

	// Build the call graph
	graph := &callGraph{
		nodes:  make([]*funcNode, 0, len(functions)),
		lookup: make(map[string]*funcNode),
	}
	for _, f := range functions {
		addTocallGraph(f, graph, symtable)
	}

	for _, n := range graph.nodes {
		n.addCallHeight()
	}

	// Determine reachability using the given entry points
	if entryPoints != nil {
		for _, entry := range entryPoints {
			root := graph.lookup[entry]
			if root != nil {
				root.markReachable()
			}
		}
	}

	return graph
}

func addTocallGraph(f *funcResult, graph *callGraph, symtable map[string]*funcResult) *funcNode {
	me, ok := graph.lookup[f.name]
	if ok {
		return me
	}
	me = &funcNode{
		name: f.name,
		fn:   f,
		outs: make([]*funcNode, 0),
	}
	for _, call := range f.calls {
		child, ok := symtable[call]
		if !ok {
			// Function not found, ignore it
			continue
		}
		n := addTocallGraph(child, graph, symtable)
		me.outs = append(me.outs, n)
	}
	graph.lookup[f.name] = me
	graph.nodes = append(graph.nodes, me)
	return me
}

func (n *funcNode) addCallHeight() {
	maxChild := -1
	for _, call := range n.outs {
		call.addCallHeight()
		if call.height > maxChild {
			maxChild = call.height
		}
	}
	n.height = maxChild + 1
}

func (n *funcNode) markReachable() {
	n.reach = true
	for _, call := range n.outs {
		call.markReachable()
	}
}

func (cg *callGraph) findUnusedFuncs() []Diagnostic {
	// Recursively walk the tree to find unreachable nodes
	unusedNames := map[string]struct{}{}
	for _, f := range cg.nodes {
		checkfuncNodeUnused(f, unusedNames)
	}
	// Sort the function names
	results := make([]Diagnostic, 0, len(unusedNames))
	for fname := range unusedNames {
		results = append(results, Diagnostic{
			Category: "unused",
			Message:  fname,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Message < results[j].Message
	})
	return results
}

func checkfuncNodeUnused(node *funcNode, unusedNames map[string]struct{}) {
	if !node.reach {
		// TODO(dustmop): Copy the position of the function definition
		unusedNames[node.name] = struct{}{}
	}
	for _, call := range node.outs {
		checkfuncNodeUnused(call, unusedNames)
	}
}

// String creates a string representation of functions in the call graph
func (cg *callGraph) String() string {
	text := ""
	for _, n := range cg.nodes {
		text += stringifyNode(n, 0)
	}
	return text
}

func stringifyNode(n *funcNode, depth int) string {
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
