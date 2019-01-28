package repo

import (
	"fmt"
	"sync"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsgraph"
)

var walkParallelism = 4

// Graph generates a map of all paths on this repository pointing
// to dsgraph.Node structs with all links configured. This is potentially
// expensive to calculate. Best to do some caching.
func Graph(r Repo) (map[string]*dsgraph.Node, error) {
	nodes := NodeList{Nodes: map[string]*dsgraph.Node{}}
	root := nodes.node(dsgraph.NtNamespace, "root")
	mu := sync.Mutex{}
	err := WalkRepoDatasets(r, func(prev *dsgraph.Node) func(int, *DatasetRef, error) (bool, error) {
		return func(depth int, ref *DatasetRef, e error) (kontinue bool, err error) {
			if e != nil {
				return false, e
			}
			mu.Lock()
			ds := nodes.nodesFromDatasetRef(r, ref)
			prev.AddLinks(dsgraph.Link{From: prev, To: ds})
			prev = ds
			mu.Unlock()
			return true, nil
		}
	}(root))
	return nodes.Nodes, err
}

// DataNodes returns a map[path]bool of all raw data nodes
func DataNodes(nodes map[string]*dsgraph.Node) (ds map[string]bool) {
	ds = map[string]bool{}
	for path, node := range nodes {
		if node.Type == dsgraph.NtData {
			ds[path] = true
		}
	}
	return
}

// NodeList is a collection of nodes
type NodeList struct {
	Nodes map[string]*dsgraph.Node
}

func (nl NodeList) node(t dsgraph.NodeType, path string) *dsgraph.Node {
	if nl.Nodes[path] != nil {
		return nl.Nodes[path]
	}
	nl.Nodes[path] = &dsgraph.Node{Type: t, Path: path}
	return nl.Nodes[path]
}

func (nl NodeList) nodesFromDatasetRef(r Repo, ref *DatasetRef) *dsgraph.Node {
	root := nl.node(dsgraph.NtDataset, ref.Path)
	ds := ref.Dataset
	if ds == nil {
		return root
	}

	root.AddLinks(dsgraph.Link{
		From: root,
		To:   nl.node(dsgraph.NtData, ds.BodyPath),
	})

	if ds.PreviousPath != "" {
		root.AddLinks(dsgraph.Link{
			From: root,
			To:   nl.node(dsgraph.NtDataset, ds.PreviousPath),
		})
	}
	if ds.Commit != nil && ds.Commit.Path != "" {
		commit := &dsgraph.Node{Type: dsgraph.NtCommit, Path: ds.Commit.Path}
		root.AddLinks(dsgraph.Link{From: root, To: commit})
	}

	if ds.Transform != nil && ds.Transform.Path != "" {
		if q, err := dsfs.LoadTransform(r.Store(), ds.Transform.Path); err == nil {
			trans := nl.node(dsgraph.NtTransform, ds.Transform.Path)
			for _, ref := range q.Resources {
				trans.AddLinks(dsgraph.Link{
					From: trans,
					To:   nl.node(dsgraph.NtDataset, ref.Path),
				})
			}
			root.AddLinks(dsgraph.Link{From: root, To: trans})
		}
	}

	return root
}

// WalkRepoDatasets visits every dataset in the history of a user's namespace
// Yes, this potentially a very expensive function to call, use sparingly.
func WalkRepoDatasets(r Repo, visit func(depth int, ref *DatasetRef, err error) (bool, error)) error {
	pll := walkParallelism
	store := r.Store()
	count, err := r.RefCount()
	if err != nil {
		return err
	} else if count == 0 {
		return ErrRepoEmpty
	}

	if count < pll {
		pll = count
	}

	doSection := func(idx, pageSize int, done chan error) error {
		refs, err := r.References(pageSize, idx*pageSize)
		if err != nil {
			done <- err
			return err
		}

		for _, ref := range refs {
			ds, err := dsfs.LoadDatasetRefs(store, ref.Path)
			if err != nil {
				err = fmt.Errorf("error loading dataset: %s", err.Error())
			}
			ref.Dataset = ds.Encode()

			kontinue, err := visit(0, &ref, err)
			if err != nil {
				err = fmt.Errorf("error visiting node: %s", err.Error())
				done <- err
				return err
			}
			if !kontinue {
				break
			}

			depth := 1
			for ref.Dataset != nil && ref.Dataset.PreviousPath != "" && ref.Dataset.PreviousPath != "/" {
				ref.Path = ref.Dataset.PreviousPath

				ds, err := dsfs.LoadDatasetRefs(store, ref.Path)
				if err != nil {
					done <- err
					return err
				}
				ref.Dataset = ds.Encode()
				kontinue, err = visit(depth, &ref, err)
				if err != nil {
					done <- err
					return err
				}
				if !kontinue {
					break
				}
				depth++
			}
		}
		done <- nil
		return nil
	}

	pageSize := count / pll
	done := make(chan error, pll)
	for i := 0; i < pll; i++ {
		go doSection(i, pageSize, done)
	}

	for i := 0; i < pll; i++ {
		err := <-done
		if err != nil {
			return err
		}
	}

	go func() {
		done <- nil
	}()

	return <-done
}
