package repo

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsgraph"
)

var walkParallelism = 4

// HasPath returns true if this repo already has a reference to
// a given path.
func HasPath(r Repo, path datastore.Key) (bool, error) {
	nodes, err := r.Graph()
	if err != nil {
		return false, fmt.Errorf("error getting repo graph: %s", err.Error())
	}
	p := path.String()
	for np := range nodes {
		if p == np {
			return true, nil
		}
	}
	return false, nil
}

func DatasetForQuery(r Repo, qpath datastore.Key) (datastore.Key, error) {
	nodes, err := r.Graph()
	if err != nil {
		return datastore.NewKey(""), fmt.Errorf("error getting repo graph: %s", err.Error())
	}
	qps := qpath.String()
	qs := QueriesMap(nodes)
	for qp, dsp := range qs {
		if qp == qps {
			return dsp, nil
		}
	}
	return datastore.NewKey(""), ErrNotFound
}

// RepoGraph generates a map of all paths on this repository pointing
// to dsgraph.Node structs with all links configured. This is potentially
// expensive to calculate. Best to do some caching.
func RepoGraph(r Repo) (map[string]*dsgraph.Node, error) {
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

// QueriesMap returns a mapped subset of a list of nodes in the form:
// 		QueryHash : DatasetHash
func QueriesMap(nodes map[string]*dsgraph.Node) (qs map[string]datastore.Key) {
	qs = map[string]datastore.Key{}
	for path, node := range nodes {
		if node.Type == dsgraph.NtDataset && len(node.Links) > 0 {
			for _, l := range node.Links {
				if l.To.Type == dsgraph.NtTransform {
					qs[path] = datastore.NewKey(l.To.Path)
				}
			}
		}
	}
	return
}

// DatasetQueries returns a mapped subset of a list of nodes in the form:
// 		DatasetHash : QueryHash
func DatasetQueries(nodes map[string]*dsgraph.Node) (qs map[string]datastore.Key) {
	qs = map[string]datastore.Key{}
	for path, node := range nodes {
		if node.Type == dsgraph.NtTransform && len(node.Links) > 0 {
			for _, l := range node.Links {
				if l.To.Type == dsgraph.NtDataset {
					// qs[path] = datastore.NewKey(l.To.Path)
					qs[l.To.Path] = datastore.NewKey(path)
				}
			}
		}
	}
	return
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
	root := nl.node(dsgraph.NtDataset, ref.Path.String())
	ds := ref.Dataset
	if ds == nil {
		return root
	}

	root.AddLinks(dsgraph.Link{
		From: root,
		To:   nl.node(dsgraph.NtData, ds.Data),
	})

	if ds.Previous.String() != "/" {
		root.AddLinks(dsgraph.Link{
			From: root,
			To:   nl.node(dsgraph.NtDataset, ds.Previous.String()),
		})
	}
	if ds.Commit != nil && ds.Commit.Path().String() != "" {
		commit := &dsgraph.Node{Type: dsgraph.NtCommit, Path: ds.Commit.Path().String()}
		root.AddLinks(dsgraph.Link{From: root, To: commit})
	}

	if ds.Abstract != nil && ds.Abstract.Path().String() != "" {
		root.AddLinks(dsgraph.Link{
			From: root,
			To:   nl.node(dsgraph.NtAbstDataset, ds.Abstract.Path().String()),
		})
	}

	if ds.Transform != nil && ds.Transform.Path().String() != "" {
		if q, err := dsfs.LoadTransform(r.Store(), ds.Transform.Path()); err == nil {
			trans := nl.node(dsgraph.NtTransform, ds.Transform.Path().String())
			for _, ref := range q.Resources {
				trans.AddLinks(dsgraph.Link{
					From: trans,
					To:   nl.node(dsgraph.NtDataset, ref.Path().String()),
				})
			}
			root.AddLinks(dsgraph.Link{From: root, To: trans})
		}
	}

	if ds.AbstractTransform != nil && ds.AbstractTransform.Path().String() != "" {
		root.AddLinks(dsgraph.Link{
			From: root,
			To:   nl.node(dsgraph.NtAbstTransform, ds.AbstractTransform.Path().String()),
		})
	}

	return root
}

// WalkDatasets visits every dataset in the history of a user's namespace
// Yes, this potentially a very expensive function to call, use sparingly.
func WalkRepoDatasets(r Repo, visit func(depth int, ref *DatasetRef, err error) (bool, error)) error {
	pll := walkParallelism
	store := r.Store()
	count, err := r.NameCount()
	if err != nil {
		return err
	} else if count == 0 {
		return ErrRepoEmpty
	}

	if count < pll {
		pll = count
	}

	doSection := func(idx, pageSize int, done chan error) error {
		refs, err := r.Namespace(pageSize, idx*pageSize)
		if err != nil {
			done <- err
			return err
		}

		for _, ref := range refs {
			ref.Dataset, err = dsfs.LoadDatasetRefs(store, ref.Path)
			// TODO - remove this once loading is more consistent.
			if err != nil {
				ref.Dataset, err = dsfs.LoadDatasetRefs(store, ref.Path)
			}
			if err != nil {
				ref.Dataset, err = dsfs.LoadDatasetRefs(store, ref.Path)
			}

			kontinue, err := visit(0, ref, err)
			if err != nil {
				done <- err
				return err
			}
			if !kontinue {
				break
			}

			depth := 1
			for ref.Dataset != nil && ref.Dataset.Previous.String() != "" && ref.Dataset.Previous.String() != "/" {
				ref.Path = ref.Dataset.Previous

				// TODO - remove this horrible hack.
				if r.Store().PathPrefix() == "ipfs" {
					if !strings.HasSuffix(ref.Path.String(), "/dataset.json") {
						ref.Path = datastore.NewKey(ref.Path.String() + "/dataset.json")
					}
				}

				ref.Dataset, err = dsfs.LoadDatasetRefs(store, ref.Path)
				kontinue, err = visit(depth, ref, err)
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

	// TODO - make properly parallel
	go func() {
		refs, err := r.GetQueryLogs(1000, 0)
		if err != nil {
			done <- err
		}
		for _, ref := range refs {
			ref.Dataset, err = dsfs.LoadDatasetRefs(store, ref.Path)
			// TODO - remove this once loading is more consistent.
			if err != nil {
				ref.Dataset, err = dsfs.LoadDatasetRefs(store, ref.Path)
			}
			if err != nil {
				ref.Dataset, err = dsfs.LoadDatasetRefs(store, ref.Path)
			}

			kontinue, err := visit(0, ref, err)
			if err != nil {
				done <- err
				return
			}
			if !kontinue {
				break
			}
		}
		done <- nil
	}()

	return <-done
}
