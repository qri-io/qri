package repo

// import (
// 	"fmt"
// 	"github.com/qri-io/dataset/dsfs"
// 	"github.com/qri-io/dataset/dsgraph"
// )

// var walkParallelism = 4

// func RepoGraph(r Repo) (*dsgraph.Node, error) {
// 	root := &dsgraph.Node{Type: dsgraph.NtNamespace, Path: "root"}
// 	err := WalkRepoDatasets(r, func(prev *dsgraph.Node) func(int, *DatasetRef, error) (bool, error) {
// 		return func(depth int, ref *DatasetRef, e error) (kontinue bool, err error) {
// 			if e != nil {
// 				return false, e
// 			}

// 			ds := NodesFromDatasetRef(ref)
// 			if depth == 0 {
// 				prev.AddLinks(dsgraph.Link{Type: dsgraph.LtNamespaceTip, From: prev, To: ds})
// 			} else {
// 				prev.AddLinks(dsgraph.Link{Type: dsgraph.LtPrevious, From: prev, To: ds})
// 			}
// 			prev = ds
// 			return true, nil
// 		}
// 	}(root))
// 	return root, err
// }

// func NodesFromDatasetRef(ref *DatasetRef) *dsgraph.Node {
// 	root := &dsgraph.Node{Type: dsgraph.NtDataset, Path: ref.Path.String()}
// 	ds := ref.Dataset
// 	if ds == nil {
// 		return root
// 	}

// 	data := &dsgraph.Node{Type: dsgraph.NtData, Path: ds.Data.Path().String()}
// 	prev := &dsgraph.Node{Type: dsgraph.NtDataset, Path: ds.Previous.Path().String()}
// 	root.AddLinks(
// 		dsgraph.Link{Type: dsgraph.LtDsData, From: root, To: data},
// 		dsgraph.Link{Type: dsgraph.LtPrevious, From: root, To: prev},
// 	)
// 	// if ds.Commit.Path().String() != "" {
// 	//   commit := &dsgraph.Node{Type: dsgraph.NtCommit, Path: ds.Commit.Path()}
// 	// root.AddLinks(dsgraph.Link{Type: dsgraph.LtDsData, From: root, To: data})
// 	// }
// 	if ds.AbstractStructure != nil && ds.AbstractStructure.Path().String() != "" {
// 		abst := &dsgraph.Node{Type: dsgraph.NtAbstStructure, Path: ds.AbstractStructure.Path().String()}
// 		root.AddLinks(dsgraph.Link{Type: dsgraph.LtAbstStructure, From: root, To: abst})
// 	}
// 	if ds.Query != nil && ds.Query.Path().String() != "" {
// 		query := &dsgraph.Node{Type: dsgraph.NtQuery, Path: ds.Query.Path().String()}
// 		root.AddLinks(dsgraph.Link{Type: dsgraph.LtQuery, From: root, To: query})
// 	}

// 	return root
// }

// // WalkDatasets visits every dataset in the history of a user's namespace
// // Yes, this potentially a very expensive function to call, use sparingly.
// func WalkRepoDatasets(r Repo, visit func(logdepth int, ref *DatasetRef, err error) (bool, error)) error {
// 	store := r.Store()
// 	count, err := r.NameCount()
// 	if err != nil {
// 		return err
// 	} else if count == 0 {
// 		return ErrRepoEmpty
// 	}

// 	if count < walkParallelism {
// 		walkParallelism = count
// 	}

// 	doSection := func(idx, pageSize int, done chan error) {
// 		refs, err := r.Namespace(pageSize, idx*pageSize)
// 		if err != nil {
// 			done <- err
// 			return
// 		}

// 		for _, ref := range refs {
// 			fmt.Println(ref.Path.String())
// 			ref.Dataset, err = dsfs.LoadDatasetRefs(store, ref.Path)
// 			kontinue, err := visit(0, ref, err)
// 			if err != nil {
// 				fmt.Println("top", err.Error())
// 				done <- err
// 				return
// 			}
// 			if !kontinue {
// 				break
// 			}

// 			depth := 1
// 			for ref.Dataset != nil && ref.Dataset.Previous.String() != "" && ref.Dataset.Previous.String() != "/" {
// 				ref.Path = ref.Dataset.Previous
// 				ref.Dataset, err = dsfs.LoadDatasetRefs(store, ref.Path)
// 				kontinue, err = visit(depth, ref, err)
// 				if err != nil {
// 					fmt.Println("prev", err.Error())
// 					done <- err
// 					return
// 				}
// 				if !kontinue {
// 					break
// 				}
// 				depth++
// 			}
// 		}
// 	}

// 	pageSize := count / walkParallelism
// 	done := make(chan error, 0)
// 	for i := 0; i < walkParallelism; i++ {
// 		go doSection(i, pageSize, done)
// 	}

// 	for i := 0; i < walkParallelism; i++ {
// 		err := <-done
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }
