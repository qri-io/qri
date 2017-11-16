package graph

import (
	"sync"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo"
)

var walkParallelism = 4

func QueriesMap(g *dsgraph.Node) (qs map[string]datastore.Key) {
	qs = map[string]datastore.Key{}
	dsgraph.Walk(g, 0, func(n *dsgraph.Node) error {
		if n.Type == dsgraph.NtDataset && len(n.Links) > 0 {
			for _, l := range n.Links {
				if l.To.Type == dsgraph.NtQuery {
					qs[l.To.Path] = datastore.NewKey(n.Path)
				}
			}
		}
		return nil
	})
	return
}

func DataNodes(g *dsgraph.Node) (ds map[string]bool) {
	ds = map[string]bool{}
	dsgraph.Walk(g, 0, func(n *dsgraph.Node) error {
		if n.Type == dsgraph.NtData {
			ds[n.Path] = true
		}
		return nil
	})
	return
}

func RepoGraph(r repo.Repo) (*dsgraph.Node, error) {
	mu := sync.Mutex{}
	root := &dsgraph.Node{Type: dsgraph.NtNamespace, Path: "root"}
	err := WalkRepoDatasets(r, func(prev *dsgraph.Node) func(int, *repo.DatasetRef, error) (bool, error) {
		return func(depth int, ref *repo.DatasetRef, e error) (kontinue bool, err error) {
			if e != nil {
				return false, e
			}

			ds := NodesFromDatasetRef(r, ref)
			mu.Lock()
			if depth == 0 {
				prev.AddLinks(dsgraph.Link{From: prev, To: ds})
			} else {
				prev.AddLinks(dsgraph.Link{From: prev, To: ds})
			}
			prev = ds
			mu.Unlock()
			return true, nil
		}
	}(root))
	return root, err
}

func NodesFromDatasetRef(r repo.Repo, ref *repo.DatasetRef) *dsgraph.Node {
	root := &dsgraph.Node{Type: dsgraph.NtDataset, Path: ref.Path.String()}
	ds := ref.Dataset
	if ds == nil {
		return root
	}

	data := &dsgraph.Node{Type: dsgraph.NtData, Path: ds.Data.String()}
	root.AddLinks(dsgraph.Link{From: root, To: data})

	if ds.Previous.Path().String() != "/" {
		prev := &dsgraph.Node{Type: dsgraph.NtDataset, Path: ds.Previous.Path().String()}
		root.AddLinks(dsgraph.Link{From: root, To: prev})
	}
	// if ds.Commit.Path().String() != "" {
	//   commit := &dsgraph.Node{Type: dsgraph.NtCommit, Path: ds.Commit.Path()}
	// root.AddLinks(dsgraph.Link{From: root, To: data})
	// }
	if ds.AbstractStructure != nil && ds.AbstractStructure.Path().String() != "" {
		abst := &dsgraph.Node{Type: dsgraph.NtAbstStructure, Path: ds.AbstractStructure.Path().String()}
		root.AddLinks(dsgraph.Link{From: root, To: abst})
	}
	if ds.Query != nil && ds.Query.Path().String() != "" {
		if q, err := dsfs.LoadQuery(r.Store(), ds.Query.Path()); err == nil {
			query := &dsgraph.Node{Type: dsgraph.NtQuery, Path: ds.Query.Path().String()}
			if q.Abstract.Path().String() != "" {
				abs := &dsgraph.Node{Type: dsgraph.NtAbstQuery, Path: q.Abstract.Path().String()}
				query.AddLinks(dsgraph.Link{From: query, To: abs})
			}
			for _, ref := range q.Resources {
				query.AddLinks(dsgraph.Link{
					From: query,
					To:   &dsgraph.Node{Type: dsgraph.NtDataset, Path: ref.Path().String()},
				})
			}
			root.AddLinks(dsgraph.Link{From: root, To: query})
		}
	}

	return root
}

// WalkDatasets visits every dataset in the history of a user's namespace
// Yes, this potentially a very expensive function to call, use sparingly.
func WalkRepoDatasets(r repo.Repo, visit func(depth int, ref *repo.DatasetRef, err error) (bool, error)) error {
	pll := walkParallelism
	store := r.Store()
	count, err := r.NameCount()
	if err != nil {
		return err
	} else if count == 0 {
		return repo.ErrRepoEmpty
	}

	if count < pll {
		pll = count
	}

	doSection := func(idx, pageSize int, done chan error) {
		refs, err := r.Namespace(pageSize, idx*pageSize)
		if err != nil {
			done <- err
			return
		}

		for _, ref := range refs {
			ref.Dataset, err = dsfs.LoadDatasetRefs(store, ref.Path)
			kontinue, err := visit(0, ref, err)
			if err != nil {
				done <- err
				return
			}
			if !kontinue {
				break
			}

			depth := 1
			for ref.Dataset != nil && ref.Dataset.Previous.String() != "" && ref.Dataset.Previous.String() != "/" {
				ref.Path = ref.Dataset.Previous
				ref.Dataset, err = dsfs.LoadDatasetRefs(store, ref.Path)
				kontinue, err = visit(depth, ref, err)
				if err != nil {
					done <- err
					return
				}
				if !kontinue {
					break
				}
				depth++
			}
		}
		done <- nil
	}

	pageSize := count / pll
	done := make(chan error, 0)
	for i := 0; i < pll; i++ {
		go doSection(i, pageSize, done)
	}

	for i := 0; i < pll; i++ {
		err := <-done
		if err != nil {
			return err
		}
	}

	return nil
}
