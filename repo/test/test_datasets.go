package test

// import (
// 	"fmt"
// 	"github.com/ipfs/go-datastore"

// 	"github.com/qri-io/cafs"
// 	"github.com/qri-io/qri/repo"
// )

// func runTestDatasets(r repo.Repo) error {
// 	for _, test := range []RepoTestFunc{
// 		testBlankName,
// 		testNames,
// 		testNamespace,
// 	} {
// 		if err := test(r); err != nil {
// 			return fmt.Errorf("RunTestNamespace: %s", err.Error())
// 		}
// 	}

// 	return nil
// }

// func testBlankName(r repo.Repo) error {
// 	err := r.PutName("", datastore.NewKey("/path/to/a/thing"))
// 	if err != repo.ErrNameRequired {
// 		return fmt.Errorf("attempting to place empty name in namestore should return repo.ErrNameRequired")
// 	}
// 	return nil
// }

// func testNames(r repo.Repo) error {
// 	name := "test"
// 	path, err := r.Store().Put(cafs.NewMemfileBytes("test", []byte(`{ "title": "test data" }`)), true)
// 	if err != nil {
// 		return fmt.Errorf("error putting test file in datastore: %s", err.Error())
// 	}

// 	if err := r.PutName(name, path); err != nil {
// 		return fmt.Errorf("repo.PutName: %s", err.Error())
// 	}

// 	resname, err := r.GetName(path)
// 	if err != nil {
// 		return fmt.Errorf("repo.GetName: %s, path: %s", err.Error(), path)
// 	}
// 	if resname != name {
// 		return fmt.Errorf("repo.GetName response mistmatch. expected: %s, got: %s", name, resname)
// 	}

// 	respath, err := r.GetPath(name)
// 	if err != nil {
// 		return fmt.Errorf("repo.GetPath: %s, name: %s", err.Error(), name)
// 	}
// 	if !respath.Equal(path) {
// 		return fmt.Errorf("repo.GetPath response mismatch. expected: %s, got: %s", path, respath)
// 	}

// 	if err := r.DeleteName(name); err != nil {
// 		return fmt.Errorf("repo.DeleteName: %s", err.Error())
// 	}

// 	if err := r.Store().Delete(path); err != nil {
// 		return fmt.Errorf("error removing file from store")
// 	}
// 	return nil
// }

// func testNamespace(r repo.Repo) error {
// 	aname := "test_namespace_a"
// 	bname := "test_namespace_b"
// 	refs := []*repo.DatasetRef{
// 		&repo.DatasetRef{Name: aname},
// 		&repo.DatasetRef{Name: bname},
// 		&repo.DatasetRef{Name: "test_namespace_c"},
// 		&repo.DatasetRef{Name: "test_namespace_d"},
// 		&repo.DatasetRef{Name: "test_namespace_e"},
// 	}
// 	for _, ref := range refs {
// 		path, err := r.Store().Put(cafs.NewMemfileBytes("test", []byte(fmt.Sprintf(`{ "title": "test_dataset_%s" }`, ref.Name))), true)
// 		if err != nil {
// 			return fmt.Errorf("error putting test file in datastore: %s", err.Error())
// 		}
// 		ref.Path = path
// 		if err := r.PutName(ref.Name, ref.Path); err != nil {
// 			return fmt.Errorf("error putting name in repo for namespace test: %s", err.Error())
// 		}
// 	}

// 	count, err := r.NameCount()
// 	if err != nil {
// 		return fmt.Errorf("repo.NameCount: %s", err.Error())
// 	}
// 	if count < 5 {
// 		return fmt.Errorf("repo.NameCount should have returned at least 5 results")
// 	}

// 	names := []*repo.DatasetRef{}
// 	pages := count
// 	pageSize := count / pages
// 	for i := 0; i <= pages; i++ {
// 		res, err := r.Namespace(pageSize, i*pageSize)
// 		if err != nil {
// 			return fmt.Errorf("repo.Namespace(%d,%d): %s", pageSize, i*pageSize, err.Error())
// 		}
// 		names = append(names, res...)
// 	}
// 	if len(names) != count {
// 		return fmt.Errorf("failed to read all paginated names. expected %d results, got %d", count, len(names))
// 	}

// 	idxs := map[string]int{}
// 	for i, ref := range names {
// 		idxs[ref.Name] = i
// 	}
// 	for i, ref := range refs {
// 		if i > 0 {
// 			if idxs[ref.Name] < idxs[refs[i-1].Name] {
// 				return fmt.Errorf("expected results to be returned in lexographical order. %s:%d, %s:%d", ref.Name, idxs[ref.Name], refs[i-1].Name, idxs[refs[i-1].Name])
// 			}
// 		}
// 	}

// 	for _, ref := range refs {
// 		if err := r.Store().Delete(ref.Path); err != nil {
// 			return fmt.Errorf("error removing path from repo store: %s", err.Error())
// 		}
// 		if err := r.DeleteName(ref.Name); err != nil {
// 			return fmt.Errorf("error removing test name from namespace: %s", err.Error())
// 		}
// 	}
// 	return nil
// }
