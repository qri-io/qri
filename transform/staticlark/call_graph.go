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

// buildCallGraph iterates the function nodes provided, and adds
// the list of calls that each makes, forming an acyclic graph
// of the entire script. This is the basis of whole file analysis,
// such as dataflow analysis
func buildCallGraph(functions []*funcNode, entryPoints []string, symtable map[string]*funcNode) *callGraph {
	// Add top level functions to the symbol table
	for _, f := range functions {
		symtable[f.name] = f
	}

	// Build the call graph
	graph := &callGraph{
		nodes:  make([]*funcNode, 0, len(functions)),
		lookup: make(map[string]*funcNode),
	}
	for _, f := range functions {
		addToCallGraph(f, graph, symtable)
	}

	for _, n := range graph.nodes {
		n.setCallHeight()
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

func addToCallGraph(f *funcNode, graph *callGraph, symtable map[string]*funcNode) *funcNode {
	me, ok := graph.lookup[f.name]
	if ok {
		return me
	}
	me = &funcNode{
		name:   f.name,
		params: f.params,
		body:   f.body,
		calls:  make([]*funcNode, 0),
	}
	for _, name := range f.callNames {
		child, ok := symtable[name]
		if !ok {
			log.Debugw("addToCallGraph func not found", "name", name)
			continue
		}
		n := addToCallGraph(child, graph, symtable)
		me.calls = append(me.calls, n)
	}
	graph.lookup[f.name] = me
	graph.nodes = append(graph.nodes, me)
	return me
}

func (n *funcNode) setCallHeight() {
	maxChild := -1
	for _, call := range n.calls {
		call.setCallHeight()
		if call.height > maxChild {
			maxChild = call.height
		}
	}
	n.height = maxChild + 1
}

func (n *funcNode) markReachable() {
	n.reach = true
	for _, call := range n.calls {
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
	for _, call := range node.calls {
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
	for _, call := range n.calls {
		if _, ok := seen[call.name]; ok {
			continue
		}
		seen[call.name] = struct{}{}
		text += stringifyNode(call, depth+1)
	}
	return text
}
