package startf

import (
	"fmt"
	"strings"
)

type CallGraph struct {
	root   *FuncNode
	nodes  []*FuncNode
	lookup map[string]*FuncNode
}

type FuncNode struct {
	name   string
	fn     *FuncResult
	outs   []*FuncNode
	reach  bool
	height int
}

func buildCallGraph(functions []*FuncResult) *CallGraph {
	symtable := map[string]*FuncResult{}
	for _, f := range functions {
		symtable[f.name] = f
	}
	symtable["print"] = &FuncResult{
		name: "print",
	}

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

	graph.root = graph.lookup["transform"]
	if graph.root != nil {
		markReachable(graph.root)
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
			fmt.Printf("not found: %s\n", call)
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

func displayCallGraph(graph *CallGraph) {
	fmt.Printf("Call Graph...\n")
	for _, f := range graph.nodes {
		displayFuncNode(f, 0)
	}
}

func displayFuncNode(node *FuncNode, depth int) {
	padding := strings.Repeat("  ", depth)
	extra := ""
	if !node.reach {
		extra = " *** DEAD CODE"
	}
	fmt.Printf("%s%s  h:%d%s\n", padding, node.name, node.height, extra)
	for _, call := range node.outs {
		displayFuncNode(call, depth+1)
	}
}
